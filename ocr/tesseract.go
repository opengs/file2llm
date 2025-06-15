package ocr

import "path"

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
	// List of language codes that should be recognized. More languages - more processing time. Order matters. Primary language has to go first as it will act as fallback. By default it will be ["eng"]
	Languages []string `json:"languages"`
	// Model to use while running tesseract. Default is `TesseractModelNormal`. Works only if `LoadCustomModels` option is set to True.
	ModelType TesseractModelType `json:"modelType"`
	// Load latest models from internet. If this is not selected, you have to manually install additional tesseract packages with models for specified languages.
	LoadCustomModels bool `json:"loadCustomModels"`
	// On startup, tesseract will download models from the internet and save them to specified location. Default is `./data/ocr/tesseract`
	ModelsFolder string `json:"modelsFolder"`
	// Variable to pass on tesseract initialization. For example you can pass {"load_system_dawg":"0"} to disable loading words list from the system
	//
	// Default is {"load_system_dawg": "0", "load_freq_dawg": "0", "load_punc_dawg": "0", "load_number_dawg": "0", "load_unambig_dawg": "0", "load_bigram_dawg": "0"}
	Variables map[string]string `json:"variables"`
	// Image formats supported by tessecart. Tesseract requires you to install third party libraries on the target machine to support all the image formats.
	// If you cant do this, you can redefine this list of supported libraries so images will be automatically converted into required format internally.
	// image/png is the only required format that must be supported and cannot be disabled.
	// Check supported formats here `https://tesseract-ocr.github.io/tessdoc/InputFormats.html`
	//
	// Default value is ["image/png", "image/jpeg", "image/tiff", "image/pnm", "image/gif", "image/webp"]. It atomatically supports image/file2llm-raw-bgra, whether you specify it or not.
	// Tesseract doesnt support compressed "image/bmp" image type. So its better to transcode it to PNG.
	SupportedImageFormats []string `json:"supportedImageFormats"`
}

func DefaultTesseractConfig() TesseractConfig {
	return TesseractConfig{
		Languages:        []string{"eng"},
		ModelType:        TesseractModelNormal,
		LoadCustomModels: false,
		ModelsFolder:     path.Join("data", "ocr", "tesseract"),
		Variables: map[string]string{
			"load_system_dawg":  "0",
			"load_freq_dawg":    "0",
			"load_punc_dawg":    "0",
			"load_number_dawg":  "0",
			"load_unambig_dawg": "0",
			"load_bigram_dawg":  "0",
		},
		SupportedImageFormats: []string{"image/png", "image/jpeg", "image/tiff", "image/pnm", "image/gif", "image/webp"},
	}
}
