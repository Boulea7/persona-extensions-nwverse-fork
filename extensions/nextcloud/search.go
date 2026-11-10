package main

import "encoding/xml"

type searchRequest struct {
	XMLName xml.Name `xml:"d:searchrequest"`
	XmlnsD  string   `xml:"xmlns:d,attr"`
	XmlnsOC string   `xml:"xmlns:oc,attr"`
	Search  struct {
		Select struct {
			Prop struct {
				FileID        xml.Name `xml:"oc:fileid"`
				DisplayName   xml.Name `xml:"d:displayname"`
				ContentType   xml.Name `xml:"d:getcontenttype"`
				ContentLength xml.Name `xml:"d:getcontentlength"`
				LastModified  xml.Name `xml:"d:getlastmodified"`
				ETag          xml.Name `xml:"d:getetag"`
				ResourceType  xml.Name `xml:"d:resourcetype"`
			} `xml:"d:prop"`
		} `xml:"d:select"`
		From struct {
			Scope struct {
				Href  string `xml:"d:href"`
				Depth string `xml:"d:depth"`
			} `xml:"d:scope"`
		} `xml:"d:from"`
		Where struct {
			And struct {
				Not struct {
					IsCollection xml.Name `xml:"d:is-collection"`
				} `xml:"d:not"`
				Like struct {
					Prop struct {
						DisplayName xml.Name `xml:"d:displayname"`
					} `xml:"d:prop"`
					Literal string `xml:"d:literal"`
				} `xml:"d:like"`
			} `xml:"d:and"`
		} `xml:"d:where"`
		OrderBy struct {
			Order struct {
				Prop struct {
					LastModified xml.Name `xml:"d:getlastmodified"`
				} `xml:"d:prop"`
				Descending xml.Name `xml:"d:descending"`
			} `xml:"d:order"`
		} `xml:"d:orderby"`
		Limit struct {
			NResults int `xml:"d:nresults"`
		} `xml:"d:limit"`
	} `xml:"d:basicsearch"`
}

func searchXML(scope, query string) []byte {
	var req searchRequest
	req.XmlnsD = "DAV:"
	req.XmlnsOC = "http://owncloud.org/ns"
	req.Search.From.Scope.Href = scope
	req.Search.From.Scope.Depth = "infinity"
	req.Search.Where.And.Like.Literal = "%" + query + "%"
	req.Search.Limit.NResults = 50
	data, err := xml.Marshal(req)
	if err != nil {
		panic(err)
	}
	return append([]byte(xml.Header), data...)
}
