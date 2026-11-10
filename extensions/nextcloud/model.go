package main

import "encoding/xml"

type fileEntry struct {
	Href          string `xml:"href"`
	DisplayName   string `xml:"propstat>prop>displayname"`
	ContentType   string `xml:"propstat>prop>getcontenttype"`
	ContentLength int64  `xml:"propstat>prop>getcontentlength"`
	LastModified  string `xml:"propstat>prop>getlastmodified"`
	ETag          string `xml:"propstat>prop>getetag"`
	FileID        int64  `xml:"propstat>prop>fileid"`
	IsCollection  bool   `xml:"propstat>prop>resourcetype>collection"`
}

type multistatus struct {
	XMLName   xml.Name    `xml:"multistatus"`
	Responses []fileEntry `xml:"response"`
}
