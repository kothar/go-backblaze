# go-backblaze
A golang client for Backblaze's B2 storage

WIP! I'd need a B2 account to really test things out.

Ideally it should behave something like this:

    client := backblaze.NewClient(accountId, applicationKey)
    bucket := client.CreateBucket("kittens", AllPublic)
    file := bucket.UploadFile(someFileBlob)
    allFiles := bucket.ListAllFiles()
