package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dr-jts/pg_featureserv/api"
	"github.com/dr-jts/pg_featureserv/config"
	"github.com/dr-jts/pg_featureserv/ui"
)

const (
	varCollectionID = "cid"
	varFeatureID    = "fid"
)

func handleRootJSON(w http.ResponseWriter, r *http.Request) {
	doRoot(w, r, api.FormatJSON)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	format := api.PathFormat(r.RequestURI)
	doRoot(w, r, format)
}

func doRoot(w http.ResponseWriter, r *http.Request, format string) {
	logRequest(r)
	urlBase := serveURLBase(r)

	// --- create content
	content := api.NewRootInfo(&config.Configuration)
	content.Links = linksRoot(urlBase, format)

	// --- encoding
	var encodedContent []byte
	var err error
	switch format {
	case api.FormatHTML:
		context := ui.PageContext{}
		context.UrlHome = urlPathFormat(urlBase, "", api.FormatHTML)
		context.UrlJSON = urlPathFormat(urlBase, "", api.FormatJSON)

		encodedContent, err = encodeHTML(content, context, ui.HTMLTemplate.Home)
	default:
		encodedContent, err = encodeJSON(content)
	}
	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentType(format), encodedContent)
}

func linksRoot(urlBase string, format string) []*api.Link {
	var links []*api.Link

	links = append(links, linkSelf(urlBase, "", format))
	links = append(links, linkAlt(urlBase, "", format))

	links = append(links, &api.Link{
		Href: urlPathFormat(urlBase, api.TagCollections, format),
		Rel:  api.RelData, Type: api.ContentType(format), Title: "collections"})

	return links
}

func linkSelf(urlBase string, path string, format string) *api.Link {
	return &api.Link{
		Href:  urlPathFormat(urlBase, path, format),
		Rel:   api.RelSelf,
		Type:  api.ContentType(format),
		Title: "This document as " + strings.ToUpper(format)}
}

func linkAlt(urlBase string, path string, format string) *api.Link {
	adt := altFormat(format)
	return &api.Link{
		Href:  urlPathFormat(urlBase, path, adt),
		Rel:   api.RelAlt,
		Type:  api.ContentType(adt),
		Title: "This document as " + strings.ToUpper(adt)}
}

func altFormat(format string) string {
	switch format {
	case api.FormatJSON:
		return api.FormatHTML
	case api.FormatHTML:
		return api.FormatJSON
	}
	// TODO: panic here?
	return ""
}

func handleCollections(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	format := api.PathFormat(r.RequestURI)
	urlBase := serveURLBase(r)

	colls, err := catalogInstance.Layers()
	if err != nil {
		writeError(w, "NoCollections", err.Error(), http.StatusInternalServerError)
		return
	}

	content := api.NewCollectionsInfo(colls)
	content.Links = linksCollections(urlBase, format)
	for _, coll := range content.Collections {
		coll.Links = linksCollection(coll.Name, urlBase, format)
	}

	// --- encoding
	var encodedContent []byte
	switch format {
	case api.FormatHTML:
		context := ui.PageContext{}
		context.UrlHome = urlPathFormat(urlBase, "", api.FormatHTML)
		context.UrlJSON = urlPathFormat(urlBase, api.TagCollections, api.FormatJSON)

		encodedContent, err = encodeHTML(content, context, ui.HTMLTemplate.Collections)
	default:
		encodedContent, err = encodeJSON(content)
	}
	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentType(format), encodedContent)
}

func linksCollections(urlBase string, format string) []*api.Link {
	var links []*api.Link
	links = append(links, linkSelf(urlBase, api.TagCollections, format))
	links = append(links, linkAlt(urlBase, api.TagCollections, format))
	return links
}

func linksCollection(name string, urlBase string, format string) []*api.Link {
	path := fmt.Sprintf("%v/%v", api.TagCollections, name)
	pathItems := api.PathItems(name)

	var links []*api.Link
	links = append(links, linkSelf(urlBase, path, format))
	links = append(links, linkAlt(urlBase, path, format))

	links = append(links, &api.Link{
		Href: urlPathFormat(urlBase, pathItems, api.FormatJSON),
		Rel:  "items", Type: api.ContentTypeJSON, Title: "Features as GeoJSON"})
	links = append(links, &api.Link{
		Href: urlPathFormat(urlBase, pathItems, api.FormatHTML),
		Rel:  "items", Type: api.ContentTypeHTML, Title: "Features as HTML"})

	return links
}

func handleCollection(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	format := api.PathFormat(r.RequestURI)
	urlBase := serveURLBase(r)

	name := getRequestVar(varCollectionID, r)

	layer, err := catalogInstance.LayerByName(name)
	if err != nil {
		writeError(w, "NoCollectionFound", err.Error(), http.StatusInternalServerError)
		return
	}
	content := api.NewCollectionInfo(layer)
	content.Links = linksCollection(name, urlBase, format)

	// --- encoding
	var encodedContent []byte
	switch format {
	case api.FormatHTML:
		context := ui.PageContext{}
		context.UrlHome = urlPathFormat(urlBase, "", api.FormatHTML)
		context.UrlCollections = urlPathFormat(urlBase, api.TagCollections, api.FormatHTML)
		context.UrlCollection = urlPathFormat(urlBase, api.PathCollection(name), api.FormatHTML)
		context.UrlJSON = urlPathFormat(urlBase, api.PathCollection(name), api.FormatJSON)
		context.CollectionTitle = layer.Title

		encodedContent, err = encodeHTML(content, context, ui.HTMLTemplate.Collection)
	default:
		encodedContent, err = encodeJSON(content)
	}
	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentType(format), encodedContent)
}

func handleCollectionItems(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	// TODO: determine content from request header?
	format := api.PathFormat(r.RequestURI)
	urlBase := serveURLBase(r)

	//--- extract request parameters
	name := getRequestVar(varCollectionID, r)

	switch format {
	case api.FormatJSON:
		writeItemsJSON(w, name, urlBase)
	case api.FormatHTML:
		writeItemsHTML(w, name, urlBase)
	}
}

func writeItemsHTML(w http.ResponseWriter, name string, urlBase string) {
	//--- query data for request
	layer, err1 := catalogInstance.LayerByName(name)
	if err1 != nil {
		writeError(w, "UnableToGetFeatures", err1.Error(), http.StatusInternalServerError)
		return
	}
	features, err2 := catalogInstance.LayerFeatures(name)
	if err2 != nil {
		writeError(w, "UnableToGetFeatures", err2.Error(), http.StatusInternalServerError)
		return
	}

	//--- assemble resonse
	content := api.NewFeatureCollectionInfo(features)

	// --- encoding
	context := ui.PageContext{}
	context.UrlHome = urlPathFormat(urlBase, "", api.FormatHTML)
	context.UrlCollections = urlPathFormat(urlBase, api.TagCollections, api.FormatHTML)
	context.UrlCollection = urlPathFormat(urlBase, api.PathCollection(name), api.FormatHTML)
	context.UrlItems = urlPathFormat(urlBase, api.PathItems(name), api.FormatHTML)
	context.UrlJSON = urlPathFormat(urlBase, api.PathItems(name), api.FormatJSON)
	context.CollectionTitle = layer.Title
	context.UseMap = true

	encodedContent, err := encodeHTML(content, context, ui.HTMLTemplate.Items)

	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentTypeHTML, encodedContent)
}

func writeItemsJSON(w http.ResponseWriter, name string, urlBase string) {
	//--- query data for request
	features, err := catalogInstance.LayerFeatures(name)
	if err != nil {
		writeError(w, "UnableToGetFeatures", err.Error(), http.StatusInternalServerError)
		return
	}

	//--- assemble resonse
	content := api.NewFeatureCollectionInfo(features)
	content.Links = linksItems(name, urlBase, api.FormatJSON)

	// --- encoding
	var encodedContent []byte
	encodedContent, err = encodeJSON(content)

	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentTypeGeoJSON, encodedContent)
}

func linksItems(name string, urlBase string, format string) []*api.Link {
	path := api.PathItems(name)

	var links []*api.Link
	links = append(links, linkSelf(urlBase, path, format))
	links = append(links, linkAlt(urlBase, path, format))

	return links
}

func handleConformance(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	// TODO: determine content from request header?
	format := api.PathFormat(r.RequestURI)
	urlBase := serveURLBase(r)

	content := api.GetConformance()

	// --- encoding
	var err error
	var encodedContent []byte
	switch format {
	case api.FormatHTML:
		context := ui.PageContext{}
		context.UrlHome = urlPathFormat(urlBase, "", api.FormatHTML)
		context.UrlJSON = urlPathFormat(urlBase, api.TagConformance, api.FormatJSON)

		encodedContent, err = encodeHTML(content, context, ui.HTMLTemplate.Conformance)
	default:
		encodedContent, err = encodeJSON(content)
	}
	if err != nil {
		writeError(w, "EncodingError", err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, api.ContentType(format), encodedContent)
}