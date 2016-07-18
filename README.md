# s3-cli -- Go version of s3cmd

Command line utility frontend to the [AWS Go SDK](http://docs.aws.amazon.com/sdk-for-go/api/)
for S3.  Inspired by [s3cmd](https://github.com/s3tools/s3cmd) and attempts to be a
drop-in replacement. 

## Features

* Compatible with [s3cmd](https://github.com/s3tools/s3cmd)'s config file
* Supports a subset of s3cmd's commands and parameters
  - including `put`, `get`, `del`, `ls`, `sync`, `cp`, `mv`
  - commands are much smarter (get, put, cp - can move to and from S3)
* When syncing directories, instead of uploading one file at a time, it 
  uploads many files in parallel resulting in more bandwidth.
* Uses multipart uploads for large files and uploads each part in parallel. This is
  accomplished using the s3manager that comes with the SDK
* More efficent at using CPU and resources on your local machine

## Install

`go get github.com/koblas/s3-cli`

## Configuration

s3-cli is compatible with s3cmd's config file, so if you already have that
configured, you're all set. Otherwise you can put this in `~/.s3cfg`:

```ini
[default]
access_key = foo
secret_key = bar
```

You can also point it to another config file with e.g. `$ s3-cli --config /path/to/s3cmd.conf ...`.

## Documentation

### cp

Copy files to and from S3

Example:

```
s3-cli cp /path/to/file s3://bucket/key/on/s3
s3-cli cp s3://bucket/key/on/s3 /path/to/file
s3-cli cp s3://bucket/key/on/s3 s3://another-bucket/some/thing
```

### get

Download a file from S3 -- really an alias for `cp`

### put

Upload a file to S3 -- really an alias for `cp`

### del

Deletes an object or a directory on S3.

Example:

```
s3-cli del [--recursive] s3://bucket/key/on/s3/
```

### rm

Alias for `del`

```
s3-cli rm [--recursive] s3://bucket/key/on/s3/
```

### sync

Sync a local directory to S3

```
s3-cli sync [--delete-removed] /path/to/folder/ s3://bucket/key/on/s3/
```

### mv

Move an object which is already on S3.

Example:

```
s3-cli mv s3://sourcebucket/source/key s3://destbucket/dest/key
```

### General Notes about s3cmd commpatability

DONE - 

* s3cmd mb s3://BUCKET
* s3cmd rb s3://BUCKET
* s3cmd ls [s3://BUCKET[/PREFIX]]
* s3cmd la
* s3cmd put FILE [FILE...] s3://BUCKET[/PREFIX]
* s3cmd get s3://BUCKET/OBJECT LOCAL_FILE
* s3cmd del s3://BUCKET/OBJECT
* s3cmd rm s3://BUCKET/OBJECT
* s3cmd du [s3://BUCKET[/PREFIX]]
* s3cmd cp s3://BUCKET1/OBJECT1 s3://BUCKET2[/OBJECT2]
* s3cmd modify s3://BUCKET1/OBJECT
* s3cmd sync LOCAL_DIR s3://BUCKET[/PREFIX] or s3://BUCKET[/PREFIX] LOCAL_DIR

TODO - for full compatibility (with s3cmd)

* s3cmd restore s3://BUCKET/OBJECT
* s3cmd info s3://BUCKET[/OBJECT]
* s3cmd mv s3://BUCKET1/OBJECT1 s3://BUCKET2[/OBJECT2]

* s3cmd setacl s3://BUCKET[/OBJECT]
* s3cmd setpolicy FILE s3://BUCKET
* s3cmd delpolicy s3://BUCKET
* s3cmd setcors FILE s3://BUCKET
* s3cmd delcors s3://BUCKET
* s3cmd payer s3://BUCKET
* s3cmd multipart s3://BUCKET [Id]
* s3cmd abortmp s3://BUCKET/OBJECT Id
* s3cmd listmp s3://BUCKET/OBJECT Id
* s3cmd accesslog s3://BUCKET
* s3cmd sign STRING-TO-SIGN
* s3cmd signurl s3://BUCKET/OBJECT <expiry_epoch|+expiry_offset>
* s3cmd fixbucket s3://BUCKET[/PREFIX]
* s3cmd ws-create s3://BUCKET
* s3cmd ws-delete s3://BUCKET
* s3cmd ws-info s3://BUCKET
* s3cmd expire s3://BUCKET
* s3cmd setlifecycle FILE s3://BUCKET
* s3cmd dellifecycle s3://BUCKET
* s3cmd cflist
* s3cmd cfinfo [cf://DIST_ID]
* s3cmd cfcreate s3://BUCKET
* s3cmd cfdelete cf://DIST_ID
* s3cmd cfmodify cf://DIST_ID
* s3cmd cfinvalinfo cf://DIST_ID[/INVAL_ID]
