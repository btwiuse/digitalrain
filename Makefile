.PHONY: all gopherjs wasm tidy clean

all: tidy gopherjs wasm

tidy:
	go mod tidy

gopherjs:
	gopherjs build .

wasm:
	bash ./wasm/build.sh

clean:
	rm -f digitalrain.js digitalrain.js.map
	rm -f wasm/digitalrain.wasm wasm/wasm_exec.js
