package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

type response struct {
    Body struct {
        Storage struct {
            Value string `json:"value"`
        } `json:"storage"`
    } `json:"body"`
}

func main() {
    var resp response
    if err := json.NewDecoder(os.Stdin).Decode(&resp); err != nil {
        panic(err)
    }
    handler := confluence.NewContentHandler()
    md, err := handler.ConvertToMarkdown(resp.Body.Storage.Value)
    if err != nil {
        panic(err)
    }
    fmt.Println(md)
}
