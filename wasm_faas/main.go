package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpHandler)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func httpHandler(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, "want /{modulename} prefix", http.StatusBadRequest)
		return
	}
	mod := parts[0]
	log.Printf("module %v requested with query %v", mod, req.URL.Query())

	env := map[string]string{
		"http_path":   req.URL.Path,
		"http_method": req.Method,
		"http_host":   req.Host,
		"http_query":  req.URL.Query().Encode(),
		"remote_addr": req.RemoteAddr,
	}

	modpath := fmt.Sprintf("target/%v.wasm", mod)
	log.Printf("loading module %v", modpath)
	out, err := invokeWasmModule(mod, modpath, env)
	if err != nil {
		log.Printf("error loading module %v", modpath)
		http.Error(w, "unable to find module "+modpath, http.StatusNotFound)
		return
	}

	// The module's stdout is written into the response.
	fmt.Fprint(w, out)
}

func invokeWasmModule(modname string, wasmPath string, env map[string]string) (string, error) {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)
	logFun := func(v uint32) {
		log.Printf("[%v]: %v", modname, v)
	}
	logStrFun := func(ctx context.Context, mod api.Module, ptr uint32, len uint32) {
		// Read the string from the module's exported memory.
		if bytes, ok := mod.Memory().Read(ptr, len); ok {
			log.Printf("[%v]: %v", modname, string(bytes))
		} else {
			log.Printf("[%v]: log_string: unable to read wasm memory", modname)
		}
	}
	_, err := runtime.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(logFun).
		Export("log_i32").
		NewFunctionBuilder().
		WithFunc(logStrFun).
		Export("log_string").
		Instantiate(ctx)

	if err != nil {
		return "", err
	}
	wasmObj, err := os.ReadFile(wasmPath)
	if err != nil {
		return "", err
	}

	var stdoutBuf bytes.Buffer
	config := wazero.NewModuleConfig().WithStdout(&stdoutBuf)
	for k, v := range env {
		config = config.WithEnv(k, v)
	}
	_, err = runtime.InstantiateWithConfig(ctx, wasmObj, config)
	if err != nil {
		return "", err
	}
	return stdoutBuf.String(), nil
}
