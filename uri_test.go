package main

import (
    "testing"
)

func TestURI(t *testing.T) {
    value := "s3://bucket-name/file.txt"
    uri, err := FileURINew(value)

    if err != nil || uri.Scheme != "s3" || uri.Bucket != "bucket-name" || *uri.Key() != "file.txt" {
        t.Error("error parsing ", value)
    }

    value = "xxx://bucket-name/a/b/c/file.txt"
    uri, err = FileURINew("xxx://bucket-name/a/b/c/file.txt")
    if err == nil {
        t.Error("error parsing ", value)
    }

    value = "test/of/test.txt"
    uri, err = FileURINew(value)
    if err != nil {
        t.Error("error parsing #3")
    }
    if uri.Path != "test/of/test.txt" || uri.Scheme != "file" || uri.Bucket != "" {
        t.Error("error parsing ", value)
    }
}
