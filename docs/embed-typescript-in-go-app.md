## 1. **Embed the compiled JavaScript binary**

Compile your TypeScript to a standalone executable (using Bun, Deno, or Node with pkg/nexe), then embed it:

```go
package main

import (
    _ "embed"
    "os"
    "os/exec"
    "path/filepath"
)

//go:embed dist/my-ts-binary
var tsBinary []byte

func runTSBinary() error {
    // Write embedded binary to temp location
    tmpFile := filepath.Join(os.TempDir(), "my-ts-binary")
    if err := os.WriteFile(tmpFile, tsBinary, 0755); err != nil {
        return err
    }
    defer os.Remove(tmpFile)

    // Execute it
    cmd := exec.Command(tmpFile, "arg1", "arg2")
    return cmd.Run()
}
```

## 2. **Embed bundled JavaScript and run with a runtime**

Bundle your TypeScript into a single JS file (using esbuild, rollup, etc.) and execute it with an embedded Node/Bun/Deno runtime:

```go
//go:embed dist/bundle.js
var jsBundle string

func runJS() error {
    cmd := exec.Command("node", "-e", jsBundle)
    return cmd.Run()
}
```

This requires the runtime to be available on the target system, or you'd need to embed that too.

## 3. **Use Bun's standalone executable feature**

Bun can create truly standalone executables that include the runtime. You'd compile your TS to a Bun executable, then embed that single binary in your Go app (approach #1).

## 4. **V8 embedding (most complex but powerful)**

Use a Go V8 binding like `rogchap/v8go` to directly execute JavaScript within your Go process without spawning external processes:

```go
import "rogchap.com/v8go"

ctx := v8go.NewContext()
ctx.RunScript(jsBundle, "bundle.js")
```

The best approach depends on your distribution requirements. For maximum portability and simplicity, I'd recommend option #1 with a Bun or Deno compiled executable—you get a single Go binary that contains everything.
