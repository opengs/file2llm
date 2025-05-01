package ocr

// Tesseract provider uses local instance of the Tesseract library to dynamically link to it and run OCR
const ProviderNameTesseract ProviderName = "TESSERACT"

// Model type used by Tesseract
type TesseractModelType string

// The fastest available model with low accuracy
const TesseractModelFast TesseractModelType = "FAST"

// Model that runs by default in tesseract instances
const TesseractModelNormal TesseractModelType = "NORMAL"

// Model with best quality. Requires more processing power
const TesseractModelBestQuality TesseractModelType = "BEST_QUALITY"

// Configuration for initializing Tesseract OCR provider
type TesseractConfig struct {
	// List of language codes that should be recognized. More languages - more processing time. Order matters. Primary language has to go first as it will act as fallback. By default it will be ["en"]
	Languages []string `json:"languages"`
	// Model to use while running tesseract. Default is `TesseractModelNormal`
	ModelType TesseractModelType `json:"modelType"`
	// On startup, tesseract will download models from the internet and save them to specified location. Default is `./data/ocr/tesseract`
	ModelsFolder string `json:"modelsFolder"`
	// Variable to pass on tesseract initialization. For example you can pass {"load_system_dawg":"0"} to disable loading words list from the system
	Variables map[string]string `json:"variables"`
	// Image formats supported by tessecart. Tesseract requires you to install third party libraries on the target machine to support all the image formats.
	// If you cant do this, you can redefine this list of supported libraries so images will be automatically converted into required format internally.
	// image/png is the only required format that must be supported and cannot be disabled.
	// Check supported formats here `https://tesseract-ocr.github.io/tessdoc/InputFormats.html`
	//
	// Default value is ["image/png", "image/jpeg", "image/tiff", "image/bmp", "image/pnm", "image/gif"]
	SupportedImageFormats []string `json:"supportedImageFormats"`
}
