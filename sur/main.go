package main

import (
    "embed"

    "github.com/suleymanmercan/sur/cmd"
)

//go:embed tasks/*.yaml
var taskFS embed.FS

func main() {
    cmd.SetTaskFS(taskFS)
    cmd.Execute()
}