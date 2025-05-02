//go:build test

package testdata

import _ "embed"

//go:embed file.doc
var DOC []byte

//go:embed file.docx
var DOCX []byte

//go:embed file.odt
var ODT []byte

//go:embed file.pdf
var PDF []byte

//go:embed image.png
var PNG []byte

//go:embed image.jpeg
var JPEG []byte

//go:embed image.bmp
var BMP []byte

//go:embed image.gif
var GIF []byte

//go:embed image.tiff
var TIFF []byte

//go:embed image.webp
var WEBP []byte
