**Jeremy Wall's goin mirror with some fixes**

See the original repo here: https://bitbucket.org/zaphar/goin

Goin Full Text Search for your files
====================================

Goin is a full text search indexer using
https://github.com/blevesearch/bleve for your files on disk. It can
handle plain text, many different images, as well as pdf files if the
correct utilities are installed.

It processes files based on their mime type making it fairly easy to
add support for more files in the future. It's still very much a work
in progress.

Usage
=====

Indexing:

`goin --index file.txt /path/to/directory/ another.file /another/directory`

Querying:

`goin --query +word -word \"phrase made up of multiple words\" field:word`

Full details of the query syntax can be found at: https://github.com/blevesearch/bleve/wiki/Query%20String%20Query

Help:

`goin --help` will give you an overview of the flags to tweak it's operation.

Install
=======

Dependencies
------------

goin depends on the (leptonica):"http://www.leptonica.com" and (tesseract):"https://github.com/tesseract-ocr" libraries to build.

Depending on how you install them you may have to set the CGO_CFLAGS and CGO_LDFLAGS environment variables to tell cgo where to find them.

`go doc cmd/go` will give you more information on configuring cgo for your environment.


For pdf support goin needs a few extra items. The (ImageMagick):"http://www.imagemagick.org/" convert tool as well as the (xpdf):http://www.foolabs.com/xpdf/" suite of tools. 
Goin uses these first to try to get text out of the pdf and falling back to OCR if the pdf is only images.

Installing goin
------------------
`go get bitbucket.org/zaphar/goin` will install the tool.

