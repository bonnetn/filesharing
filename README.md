# File sharing script

A simple script to share files between people.

The "uploader" will open a connection and stream the file to the server.

The server will take this stream and directly redirect it to the "downloader".

It uses the Linux syscall **splice**, so the data does not even flow through the app in userspace.
