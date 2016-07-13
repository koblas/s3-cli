# s3-cli

Go version of s3cmd

Why? Because I needed to copy about 1TB from S3 to my local machine
and noticed that s3cmd is a total pig when it comes to "sync"
operations and there really isn't any good alternatives. The only
alternative that I found was a nodejs one which while better crashed
under some cirucumstances.

This should also end up being a good example of the S3 GoLang API as well. Though the 
AWS Go Lang SDK for S3 has really good examples.

Note: This is current a work in progress, like most of the universe.

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

TODO - for full compatibility:

* s3cmd get -- handle recursive
* s3cmd del -- handle recursive

* s3cmd restore s3://BUCKET/OBJECT
* s3cmd sync LOCAL_DIR s3://BUCKET[/PREFIX] or s3://BUCKET[/PREFIX] LOCAL_DIR
* s3cmd info s3://BUCKET[/OBJECT]
* s3cmd cp s3://BUCKET1/OBJECT1 s3://BUCKET2[/OBJECT2]
* s3cmd modify s3://BUCKET1/OBJECT
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
