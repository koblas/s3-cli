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
	if uri.String() != value {
		t.Error("roundtrip ", value, uri.String())
	}

	value = "xxx://bucket-name/a/b/c/file.txt"
	uri, err = FileURINew("xxx://bucket-name/a/b/c/file.txt")
	if err == nil {
		t.Error("error parsing ", value)
	}

	value = "test/of/test.txt"
	uri, err = FileURINew(value)
	if err != nil {
		t.Error("error parsing ", value)
	}
	if uri.Path != "test/of/test.txt" || uri.Scheme != "file" || uri.Bucket != "" {
		t.Error("error parsing ", value)
	}
	if uri.String() != "file://test/of/test.txt" {
		t.Error("roundtrip ", value, uri.String())
	}

	// Test joins -- first file
	value = "test/of/test.txt"
	uri, err = FileURINew(value)
	if err != nil {
		t.Error("error parsing ", value)
	}

	if turi := uri.Join("/new/file.txt"); turi.Path != "/new/file.txt" {
		t.Error("error join /new/file.txt", turi.Path)
	}
	if turi := uri.Join("new/file.txt"); turi.Path != "test/of/new/file.txt" {
		t.Error("error join new/file.txt", turi.Path)
	}

	value = "test/of/test%2Fwith%2Fslashes.txt"
	uri, err = FileURINew(value)
	if err != nil {
		t.Error("error parsing ", value)
	}
	if uri.Path != "test/of/test%2Fwith%2Fslashes.txt" || uri.Scheme != "file" || uri.Bucket != "" {
		t.Error("error parsing ", value)
	}
	if uri.String() != "file://test/of/test%2Fwith%2Fslashes.txt" {
		t.Error("roundtrip ", value, uri.String())
	}

	value = "test/of/test%2Fwith%2Fslashes and spaces.txt"
	uri, err = FileURINew(value)
	if err != nil {
		t.Error("error parsing ", value)
	}
	if uri.Path != "test/of/test%2Fwith%2Fslashes and spaces.txt" || uri.Scheme != "file" || uri.Bucket != "" {
		t.Error("error parsing ", value)
	}
	if uri.String() != "file://test/of/test%2Fwith%2Fslashes and spaces.txt" {
		t.Error("roundtrip ", value, uri.String())
	}
}
