package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser"
	"github.com/spf13/cobra"
)

var serveCMD = &cobra.Command{
	Use:   "serve",
	Short: "Start REST API server",
	Long:  "Start REST API server and process requests with files",
	RunE: func(cmd *cobra.Command, args []string) error {
		var ocrProvider ocr.Provider
		ocrProviderType, _ := cmd.Flags().GetString("ocr-provider")
		if ocrProviderType == "TESSERACT" {
			config := ocr.DefaultTesseractConfig()
			config.Languages, _ = cmd.Flags().GetStringSlice("ocr-tesseract-languages")
			config.LoadCustomModels, _ = cmd.Flags().GetBool("ocr-tesseract-load-custom-models")
			tesseractModelType, _ := cmd.Flags().GetString("ocr-tesseract-model")
			if !slices.Contains([]string{"FAST", "NORMAL", "BEST_QUALITY"}, tesseractModelType) {
				return errors.New("tesseract model type is not supported")
			}
			config.ModelType = ocr.TesseractModelType(tesseractModelType)
			config.ModelsFolder, _ = cmd.Flags().GetString("ocr-tesseract-models-folder")
			config.SupportedImageFormats, _ = cmd.Flags().GetStringSlice("ocr-tesseract-supported-mime-types")

			tesseractPoolSize, _ := cmd.Flags().GetUint32("ocr-tesseract-pool-size")

			tesseract := ocr.NewTesseractPool(tesseractPoolSize, config)
			if err := tesseract.Init(context.Background()); err != nil {
				return fmt.Errorf("failed to initialize tesseract OCR provider: %s", err.Error())
			}
			defer tesseract.Destroy(context.Background())

			ocrProvider = tesseract
		}

		if ocrProvider == nil {
			return errors.New("unsupported ocr provider")
		}

		fileParser := parser.New(ocrProvider)

		ginEngine := gin.Default()
		ginEngine.POST("/file", func(ctx *gin.Context) {
			result := fileParser.Parse(ctx.Request.Context(), ctx.Request.Body)
			ctx.JSON(http.StatusOK, gin.H{
				"text": result.String(),
				"raw":  result,
			})
		})
		ginEngine.POST("/ocr", func(ctx *gin.Context) {
			result, err := ocrProvider.OCR(ctx.Request.Context(), ctx.Request.Body)
			var errorString string
			if err != nil {
				errorString = err.Error()
			}
			ctx.JSON(http.StatusOK, gin.H{
				"status": err == nil,
				"result": result,
				"error":  errorString,
			})
		})

		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetUint("port")
		if err := ginEngine.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			return errors.Join(errors.New("failed to run HTTP server engine"), err)
		}

		return nil
	},
}

func init() {
	serveCMD.Flags().String("host", "0.0.0.0", "Host server will be listening on")
	serveCMD.Flags().Uint("port", 8884, "Port server will be listening on")

	serveCMD.Flags().String("ocr-provider", "TESSERACT", "OCR provider to user. Possible values are TESSERACT, TESSERACT_SERVER, PADDLE")
	serveCMD.Flags().StringSlice("ocr-tesseract-languages", []string{"eng"}, "List of languages that will be used. Those languages must be preloaded and installed on the target machine")
	serveCMD.Flags().Bool("ocr-tesseract-load-custom-models", false, "Load custom OCR models for tesseract during runtime")
	serveCMD.Flags().String("ocr-tesseract-model", "NORMAL", "Model type to user. Supported values are FAST, NORMAL, BEST_QUALITY. Only works when ")
	serveCMD.Flags().String("ocr-tesseract-models-folder", "./data/ocr/tesseract", "Location on the disk where to load custom tesseract models")
	serveCMD.Flags().StringSlice("ocr-tesseract-supported-mime-types", []string{"image/png", "image/jpeg", "image/tiff", "image/pnm", "image/gif", "image/webp"}, "List of mime types supported by tesseract")
	serveCMD.Flags().Uint32("ocr-tesseract-pool-size", 1, "Maximum number of tesseract instances running at the same time")
}
