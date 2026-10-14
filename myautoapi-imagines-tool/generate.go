package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	apiURL                     = "https://api.myautoapi.com/v1/images/generations"
	apiEditURL                 = "https://api.myautoapi.com/v1/images/edits"
	apiResponsesURL            = "https://api.myautoapi.com/v1/responses"
	modelName                  = "gpt-image-2"
	flareModelName             = "gpt-image-2.5-flare"
	sunburstModelName          = "gpt-image-2.5-sunburst"
	defaultModelName           = flareModelName
	thModelName                = "gpt-image-1k-th"
	adobeModelName             = "gpt-image-2-adobe"
	grokModelName              = "grok-imagine-image"
	banana2ModelName           = "nano-banana-2"
	bananaProModelName         = "nano-banana-pro"
	defaultSize                = "1024x1024"
	defaultQuality             = "high"
	gptImage25DefaultSize      = "2048x2048"
	thDefaultSize              = "1024x1024"
	thDefaultQuality           = "high"
	adobeDefaultSize           = "1024x1536"
	adobeDefaultQuality        = "low"
	grokDefaultRatio           = "1:1"
	grokDefaultResolution      = "1k"
	bananaDefaultRatio         = "1:1"
	bananaDefaultResolution    = "1k"
	defaultOutputFormat        = "png"
	maxResponseBytes           = 100 * 1024 * 1024
	minImagePixels             = 655360
	maxImagePixels             = 8294400
	maxImageEdge               = 3840
	maxInputImageBytes         = 50 * 1024 * 1024
	maxInputImages             = 16
	maxTotalInputBytes         = 200 * 1024 * 1024
	maxBananaInputImages       = 14
	maxBananaInputBytes        = 7 * 1024 * 1024
	maxConcurrentImageRequests = 3
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type generationOptions struct {
	prompt       string
	model        string
	n            int
	size         string
	quality      string
	aspectRatio  string
	resolution   string
	outputFormat string
	outputDir    string
	outputName   string
	images       []string
}

type imageModelSpec struct {
	defaultSize         string
	defaultQuality      string
	supportedSizes      []string
	supportedQuality    []string
	supportedRatios     []string
	supportedResolution []string
	supportsEdit        bool
	maxOutputs          int
}

var supportedImageModels = []string{flareModelName, sunburstModelName, modelName, thModelName, adobeModelName, grokModelName, banana2ModelName, bananaProModelName}

var bananaAspectRatios = []string{"1:1", "3:2", "2:3", "3:4", "1:4", "4:1", "4:3", "4:5", "5:4", "1:8", "8:1", "9:16", "16:9", "21:9", "9:21"}

var imageModelSpecs = map[string]imageModelSpec{
	modelName: {
		defaultSize:      defaultSize,
		defaultQuality:   defaultQuality,
		supportedQuality: []string{"low", "medium", "high", "auto"},
		supportsEdit:     true,
	},
	flareModelName: {
		defaultSize:      gptImage25DefaultSize,
		defaultQuality:   defaultQuality,
		supportedQuality: []string{"low", "medium", "high", "xhigh", "max", "auto"},
		supportsEdit:     true,
	},
	sunburstModelName: {
		defaultSize:      gptImage25DefaultSize,
		defaultQuality:   defaultQuality,
		supportedQuality: []string{"low", "medium", "high", "xhigh", "max", "auto"},
		supportsEdit:     true,
	},
	thModelName: {
		defaultSize:      thDefaultSize,
		defaultQuality:   thDefaultQuality,
		supportedSizes:   []string{thDefaultSize},
		supportedQuality: []string{"low", "high"},
		supportsEdit:     true,
	},
	adobeModelName: {
		defaultSize:      adobeDefaultSize,
		defaultQuality:   adobeDefaultQuality,
		supportedSizes:   []string{adobeDefaultSize, "1536x2304", "2304x3456"},
		supportedQuality: []string{adobeDefaultQuality},
	},
	grokModelName: {
		supportedQuality:    []string{"low", "medium", "high"},
		supportedRatios:     []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"},
		supportedResolution: []string{"1k", "2k"},
		supportsEdit:        true,
		maxOutputs:          9,
	},
	banana2ModelName: {
		supportedRatios:     bananaAspectRatios,
		supportedResolution: []string{"512", "1k", "2k", "4k"},
		supportsEdit:        true,
	},
	bananaProModelName: {
		supportedRatios:     bananaAspectRatios,
		supportedResolution: []string{"1k", "2k", "4k"},
		supportsEdit:        true,
	},
}

var (
	dataImagePattern     = regexp.MustCompile(`(?i)data:image/[a-z0-9.+-]+;base64,[a-z0-9+/=\r\n]+`)
	markdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(<?(https://[^<>\s)]+)>?\)`)
)

type stringList []string

func (list *stringList) String() string {
	return strings.Join(*list, ",")
}

func (list *stringList) Set(value string) error {
	*list = append(*list, value)
	return nil
}

type imageInfo struct {
	format     string
	extension  string
	dimensions []int
}

func generate(arguments []string, stdout io.Writer) error {
	options, err := parseGenerationOptions(arguments)
	if err != nil {
		return err
	}
	endpoint := apiURL
	if isBananaModel(options.model) {
		endpoint = apiResponsesURL
	}
	return runGeneration(options, stdout, newHTTPClient(), endpoint)
}

func edit(arguments []string, stdout io.Writer) error {
	options, err := parseEditOptions(arguments)
	if err != nil {
		return err
	}
	endpoint := apiEditURL
	if isBananaModel(options.model) {
		endpoint = apiResponsesURL
	}
	return runEdit(options, stdout, newHTTPClient(), endpoint)
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Minute, CheckRedirect: secureRedirectPolicy}
}

func secureRedirectPolicy(request *http.Request, via []*http.Request) error {
	if request.Header.Get("Authorization") != "" {
		return errors.New("带凭据的请求不允许重定向")
	}
	for _, previous := range via {
		if previous.Header.Get("Authorization") != "" {
			return errors.New("带凭据的请求不允许重定向")
		}
	}
	if request.URL.Scheme != "https" {
		return errors.New("图片下载重定向只允许 HTTPS")
	}
	if len(via) >= 5 {
		return errors.New("图片下载重定向次数过多")
	}
	return nil
}

func parseGenerationOptions(arguments []string) (generationOptions, error) {
	return parseImageOptions("generate", arguments, false)
}

func parseEditOptions(arguments []string) (generationOptions, error) {
	return parseImageOptions("edit", arguments, true)
}

func parseImageOptions(command string, arguments []string, requireImages bool) (generationOptions, error) {
	options := generationOptions{}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.prompt, "prompt", "", "图片描述")
	flags.StringVar(&options.model, "model", defaultModelName, "模型")
	flags.IntVar(&options.n, "n", 1, "返回图片数量")
	flags.StringVar(&options.size, "size", "", "图片尺寸")
	flags.StringVar(&options.quality, "quality", "", "图片质量")
	flags.StringVar(&options.aspectRatio, "aspect-ratio", "", "图片比例")
	flags.StringVar(&options.resolution, "resolution", "", "图片分辨率档位")
	flags.StringVar(&options.outputFormat, "output-format", defaultOutputFormat, "输出格式")
	flags.StringVar(&options.outputDir, "output-dir", "", "输出目录")
	flags.StringVar(&options.outputName, "output-name", "", "输出文件名")
	flags.Var((*stringList)(&options.images), "image", "输入图片路径，可重复")
	if err := flags.Parse(arguments); err != nil {
		return options, fmt.Errorf("参数无效：%w", err)
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf("存在无法识别的参数：%s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(options.prompt) == "" {
		return options, errors.New("图片描述不能为空。")
	}
	options.prompt = strings.TrimSpace(options.prompt)
	options.model = strings.TrimSpace(options.model)
	options.size = strings.ToLower(strings.TrimSpace(options.size))
	options.quality = strings.ToLower(strings.TrimSpace(options.quality))
	options.aspectRatio = strings.ToLower(strings.TrimSpace(options.aspectRatio))
	options.resolution = strings.ToLower(strings.TrimSpace(options.resolution))
	options.outputFormat = strings.ToLower(strings.TrimSpace(options.outputFormat))
	options.outputDir = strings.TrimSpace(options.outputDir)
	options.outputName = strings.TrimSpace(options.outputName)
	modelSpec, ok := imageModelSpecs[options.model]
	if !ok {
		return options, fmt.Errorf("V0.8.1 支持的模型只有：%s", strings.Join(supportedImageModels, "、"))
	}
	provided := map[string]bool{}
	flags.Visit(func(visited *flag.Flag) { provided[visited.Name] = true })
	if options.model == grokModelName || isBananaModel(options.model) {
		if provided["size"] {
			return options, fmt.Errorf("模型 %s 使用 --aspect-ratio 和 --resolution 设置画面规格。", options.model)
		}
		if options.model == grokModelName {
			if options.aspectRatio == "" {
				options.aspectRatio = grokDefaultRatio
			}
			if options.resolution == "" {
				options.resolution = grokDefaultResolution
			}
		} else {
			if options.aspectRatio == "" {
				options.aspectRatio = bananaDefaultRatio
			}
			if options.resolution == "" {
				options.resolution = bananaDefaultResolution
			}
		}
		if requireImages && options.model == grokModelName && (provided["aspect-ratio"] || provided["resolution"] || provided["quality"]) {
			return options, errors.New("Grok Imagine 参考图和修图直接使用原图规格。")
		}
		if isBananaModel(options.model) && provided["quality"] {
			return options, errors.New("Nano Banana 使用 --resolution 和 --aspect-ratio，不使用 --quality。")
		}
	} else {
		if provided["aspect-ratio"] || provided["resolution"] {
			return options, fmt.Errorf("模型 %s 使用 --size 设置图片尺寸。", options.model)
		}
		if options.size == "" {
			options.size = modelSpec.defaultSize
		}
	}
	if options.quality == "" {
		options.quality = modelSpec.defaultQuality
	}
	maxOutputs := 10
	if modelSpec.maxOutputs > 0 {
		maxOutputs = modelSpec.maxOutputs
	}
	if options.n < 1 || options.n > maxOutputs {
		return options, fmt.Errorf("模型 %s 的图片数量必须在 1 到 %d 之间。", options.model, maxOutputs)
	}
	if requireImages && !modelSpec.supportsEdit {
		return options, fmt.Errorf("模型 %s 当前只支持文生图；参考图和修图请使用 %s。", options.model, defaultModelName)
	}
	if requireImages && len(options.images) == 0 {
		return options, errors.New("edit 必须至少提供一张 --image。")
	}
	if !requireImages && len(options.images) > 0 {
		return options, errors.New("generate 不接受 --image，请使用 edit。")
	}
	if requireImages && isBananaModel(options.model) && len(options.images) > maxBananaInputImages {
		return options, fmt.Errorf("Nano Banana 参考图最多支持 %d 张。", maxBananaInputImages)
	}
	if options.model == grokModelName {
		if !containsString(modelSpec.supportedRatios, options.aspectRatio) {
			return options, fmt.Errorf("Grok Imagine 支持的图片比例为：%s", strings.Join(modelSpec.supportedRatios, "、"))
		}
		if !containsString(modelSpec.supportedResolution, options.resolution) {
			return options, fmt.Errorf("Grok Imagine 支持的分辨率档位为：%s", strings.Join(modelSpec.supportedResolution, "、"))
		}
	} else if isBananaModel(options.model) {
		if !containsString(modelSpec.supportedRatios, options.aspectRatio) {
			return options, fmt.Errorf("模型 %s 支持的图片比例为：%s", options.model, strings.Join(modelSpec.supportedRatios, "、"))
		}
		if !containsString(modelSpec.supportedResolution, options.resolution) {
			return options, fmt.Errorf("模型 %s 支持的分辨率档位为：%s", options.model, strings.Join(modelSpec.supportedResolution, "、"))
		}
	} else if err := validateImageSize(options.size); err != nil {
		return options, err
	}
	if options.quality != "" && !isSupportedQuality(options.quality) {
		return options, errors.New("图片质量只支持 low、medium、high、xhigh、max 或 auto。")
	}
	if len(modelSpec.supportedSizes) > 0 && !containsString(modelSpec.supportedSizes, options.size) {
		return options, fmt.Errorf("模型 %s 当前已验证尺寸只有：%s", options.model, strings.Join(modelSpec.supportedSizes, "、"))
	}
	if options.quality != "" && len(modelSpec.supportedQuality) > 0 && !containsString(modelSpec.supportedQuality, options.quality) {
		return options, fmt.Errorf("模型 %s 当前已验证质量只有：%s", options.model, strings.Join(modelSpec.supportedQuality, "、"))
	}
	if options.outputFormat != defaultOutputFormat {
		return options, errors.New("当前稳定输出格式只支持 png。")
	}
	return options, nil
}

func isBananaModel(model string) bool {
	return model == banana2ModelName || model == bananaProModelName
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateImageSize(value string) error {
	if value == "auto" {
		return nil
	}
	parts := strings.Split(value, "x")
	if len(parts) != 2 {
		return errors.New("图片尺寸必须是 auto 或 WIDTHxHEIGHT。")
	}
	width, widthErr := strconv.Atoi(parts[0])
	height, heightErr := strconv.Atoi(parts[1])
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return errors.New("图片尺寸必须是 auto 或有效的 WIDTHxHEIGHT。")
	}
	if width%16 != 0 || height%16 != 0 {
		return errors.New("图片宽度和高度必须是 16 的倍数。")
	}
	if width > maxImageEdge || height > maxImageEdge {
		return fmt.Errorf("图片宽度和高度不能超过 %d。", maxImageEdge)
	}
	shortEdge, longEdge := width, height
	if shortEdge > longEdge {
		shortEdge, longEdge = longEdge, shortEdge
	}
	if longEdge > shortEdge*3 {
		return errors.New("图片长边不能超过短边的 3 倍。")
	}
	pixels := width * height
	if pixels < minImagePixels || pixels > maxImagePixels {
		return fmt.Errorf("图片总像素必须在 %d 到 %d 之间。", minImagePixels, maxImagePixels)
	}
	return nil
}

func isSupportedQuality(value string) bool {
	switch value {
	case "low", "medium", "high", "xhigh", "max", "auto":
		return true
	default:
		return false
	}
}

func runGeneration(options generationOptions, stdout io.Writer, client httpDoer, endpoint string) error {
	root, err := codexHome()
	if err != nil {
		return err
	}
	outputRoot := options.outputDir
	if outputRoot == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("无法确定默认图片目录：%w", err)
		}
		outputRoot = filepath.Join(workingDirectory, "images")
	}
	apiKey, err := loadAPIKey(root)
	if err != nil {
		return err
	}
	imageBytes, err := collectRequestedImages(options, func(requestOptions generationOptions) ([][]byte, error) {
		return requestGenerationImages(requestOptions, client, endpoint, apiKey)
	})
	if err != nil {
		return err
	}
	outputFiles, saveErr := saveOutputImages(outputRoot, options.outputName, imageBytes)
	if len(outputFiles) == 0 && saveErr != nil {
		return fmt.Errorf("无法保存图片：%w", saveErr)
	}
	summary := outputSummary(outputFiles, options.n)
	writeResult(stdout, summary)
	return nil
}

func requestGenerationImages(options generationOptions, client httpDoer, endpoint, apiKey string) ([][]byte, error) {
	if isBananaModel(options.model) {
		return requestBananaImages(options, client, endpoint, apiKey)
	}
	payload := generationPayload(options)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("无法编码图片请求。")
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, errors.New("无法创建图片请求。")
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New(redactError("MyAutoAPI 请求无法发送："+err.Error(), apiKey))
	}
	defer response.Body.Close()
	responseBody, err := readLimited(response.Body)
	if err != nil {
		return nil, fmt.Errorf("无法读取 MyAutoAPI 响应：%w", err)
	}
	return decodeImageResponse(response.StatusCode, responseBody, client, 0, apiKey)
}

func generationPayload(options generationOptions) map[string]any {
	payload := map[string]any{
		"model":  options.model,
		"prompt": options.prompt,
		"n":      options.n,
	}
	if options.model == grokModelName {
		payload["response_format"] = "b64_json"
		payload["aspect_ratio"] = options.aspectRatio
		payload["resolution"] = options.resolution
		if options.quality != "" {
			payload["quality"] = options.quality
		}
		return payload
	}
	payload["size"] = options.size
	payload["quality"] = options.quality
	payload["output_format"] = options.outputFormat
	return payload
}

func runEdit(options generationOptions, stdout io.Writer, client httpDoer, endpoint string) error {
	root, err := codexHome()
	if err != nil {
		return err
	}
	outputRoot := options.outputDir
	if outputRoot == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("无法确定默认图片目录：%w", err)
		}
		outputRoot = filepath.Join(workingDirectory, "images")
	}
	apiKey, err := loadAPIKey(root)
	if err != nil {
		return err
	}
	imageBytes, err := collectRequestedImages(options, func(requestOptions generationOptions) ([][]byte, error) {
		if isBananaModel(requestOptions.model) {
			return requestBananaImages(requestOptions, client, endpoint, apiKey)
		}
		return requestEditImages(requestOptions, client, endpoint, apiKey)
	})
	if err != nil {
		return err
	}
	outputFiles, saveErr := saveOutputImages(outputRoot, options.outputName, imageBytes)
	if len(outputFiles) == 0 && saveErr != nil {
		return fmt.Errorf("无法保存图片：%w", saveErr)
	}
	writeResult(stdout, outputSummary(outputFiles, options.n))
	return nil
}

func requestBananaImages(options generationOptions, client httpDoer, endpoint, apiKey string) ([][]byte, error) {
	payload, err := bananaPayload(options)
	if err != nil {
		return nil, err
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("无法编码 Nano Banana 请求。")
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, errors.New("无法创建 Nano Banana 请求。")
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New(redactError("MyAutoAPI Nano Banana 请求无法发送："+err.Error(), apiKey))
	}
	defer response.Body.Close()
	responseBody, err := readLimited(response.Body)
	if err != nil {
		return nil, fmt.Errorf("无法读取 MyAutoAPI Nano Banana 响应：%w", err)
	}
	return decodeResponsesImageResponse(response.StatusCode, responseBody, client, apiKey)
}

func bananaPayload(options generationOptions) (map[string]any, error) {
	content := []any{map[string]any{"type": "input_text", "text": options.prompt}}
	for _, path := range options.images {
		imageURL, err := encodeBananaInputImage(path)
		if err != nil {
			return nil, err
		}
		content = append(content, map[string]any{"type": "input_image", "image_url": imageURL})
	}
	imageSize := strings.ToUpper(options.resolution)
	if options.resolution == "512" {
		imageSize = "512"
	}
	return map[string]any{
		"model": options.model,
		"input": []any{map[string]any{
			"role":    "user",
			"content": content,
		}},
		"stream": false,
		"response_format": map[string]any{
			"type":         "image",
			"mime_type":    "image/png",
			"aspect_ratio": options.aspectRatio,
			"image_size":   imageSize,
		},
	}, nil
}

func encodeBananaInputImage(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("无法读取输入图片：%s", path)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("输入图片不是普通文件：%s", path)
	}
	if info.Size() <= 0 || info.Size() > maxBananaInputBytes {
		return "", fmt.Errorf("Nano Banana 单张参考图大小必须大于 0 且不超过 7 MiB：%s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("无法读取输入图片：%s", path)
	}
	var mimeType string
	switch inspectImage(content).format {
	case "png":
		mimeType = "image/png"
	case "jpeg":
		mimeType = "image/jpeg"
	case "webp":
		mimeType = "image/webp"
	default:
		return "", fmt.Errorf("Nano Banana 参考图必须是 PNG、JPEG 或 WebP：%s", path)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}

func collectRequestedImages(options generationOptions, request func(generationOptions) ([][]byte, error)) ([][]byte, error) {
	type requestResult struct {
		index  int
		images [][]byte
		err    error
	}

	results := make([]requestResult, options.n)
	resultChannel := make(chan requestResult, options.n)
	semaphore := make(chan struct{}, min(options.n, maxConcurrentImageRequests))
	for index := 0; index < options.n; index++ {
		go func(requestIndex int) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			requestOptions := options
			requestOptions.n = 1
			images, err := request(requestOptions)
			resultChannel <- requestResult{index: requestIndex, images: images, err: err}
		}(index)
	}
	for range options.n {
		result := <-resultChannel
		results[result.index] = result
	}

	images := make([][]byte, 0, options.n)
	var firstError error
	for _, result := range results {
		if firstError == nil && result.err != nil {
			firstError = result.err
		}
		for _, image := range result.images {
			if len(images) == options.n {
				return images, nil
			}
			images = append(images, image)
		}
	}
	if len(images) != 0 {
		return images, nil
	}
	if firstError != nil {
		return nil, firstError
	}
	return nil, errors.New("MyAutoAPI 未返回可保存的图片。")
}

func requestEditImages(options generationOptions, client httpDoer, endpoint, apiKey string) ([][]byte, error) {
	multipartFile, contentType, err := buildEditMultipart(options)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = multipartFile.Close()
		_ = os.Remove(multipartFile.Name())
	}()
	stat, err := multipartFile.Stat()
	if err != nil {
		return nil, errors.New("无法读取图片编辑请求。")
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, multipartFile)
	if err != nil {
		return nil, errors.New("无法创建图片编辑请求。")
	}
	request.ContentLength = stat.Size()
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New(redactError("MyAutoAPI 图片编辑请求无法发送："+err.Error(), apiKey))
	}
	responseBody, err := readLimited(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("无法读取 MyAutoAPI 图片编辑响应：%w", err)
	}
	return decodeImageResponse(response.StatusCode, responseBody, client, 0, apiKey)
}

func buildEditMultipart(options generationOptions) (*os.File, string, error) {
	if len(options.images) > maxInputImages {
		return nil, "", fmt.Errorf("输入图片最多支持 %d 张。", maxInputImages)
	}
	totalBytes := int64(0)
	for _, path := range options.images {
		info, err := os.Stat(path)
		if err != nil {
			return nil, "", fmt.Errorf("无法读取输入图片：%s", path)
		}
		if !info.Mode().IsRegular() {
			return nil, "", fmt.Errorf("输入图片不是普通文件：%s", path)
		}
		if info.Size() <= 0 || info.Size() > maxInputImageBytes {
			return nil, "", fmt.Errorf("输入图片大小必须在 1 到 50 MiB 之间：%s", path)
		}
		totalBytes += info.Size()
		if totalBytes > maxTotalInputBytes {
			return nil, "", errors.New("输入图片总大小不能超过 200 MiB。")
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("无法读取输入图片：%s", path)
		}
		content, readErr := io.ReadAll(io.LimitReader(file, maxInputImageBytes+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			return nil, "", fmt.Errorf("无法检查输入图片：%s", path)
		}
		if len(content) == 0 || inspectImage(content).format == "unknown" {
			return nil, "", fmt.Errorf("输入文件不是支持的图片格式：%s", path)
		}
	}
	temporaryFile, err := os.CreateTemp("", "myautoapi-image-edit-*.multipart")
	if err != nil {
		return nil, "", errors.New("无法创建临时图片编辑请求。")
	}
	removeOnError := true
	defer func() {
		if removeOnError {
			_ = temporaryFile.Close()
			_ = os.Remove(temporaryFile.Name())
		}
	}()
	writer := multipart.NewWriter(temporaryFile)
	fields := editFields(options)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", errors.New("无法编码图片编辑请求。")
		}
	}
	imageField := "image"
	if len(options.images) > 1 {
		imageField = "image[]"
	}
	for _, path := range options.images {
		part, err := writer.CreateFormFile(imageField, filepath.Base(path))
		if err != nil {
			return nil, "", errors.New("无法编码输入图片。")
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("无法读取输入图片：%s", path)
		}
		_, copyErr := io.Copy(part, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return nil, "", fmt.Errorf("无法编码输入图片：%s", path)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", errors.New("无法完成图片编辑请求。")
	}
	if _, err := temporaryFile.Seek(0, io.SeekStart); err != nil {
		return nil, "", errors.New("无法准备图片编辑请求。")
	}
	removeOnError = false
	return temporaryFile, writer.FormDataContentType(), nil
}

func editFields(options generationOptions) map[string]string {
	fields := map[string]string{
		"model":  options.model,
		"prompt": options.prompt,
		"n":      strconv.Itoa(options.n),
	}
	if options.model == grokModelName {
		fields["response_format"] = "b64_json"
		return fields
	}
	fields["size"] = options.size
	fields["quality"] = options.quality
	fields["output_format"] = options.outputFormat
	return fields
}

func decodeImageResponse(statusCode int, responseBody []byte, client httpDoer, expectedCount int, apiKey string) ([][]byte, error) {
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("MyAutoAPI 返回 HTTP %d。", statusCode)
	}
	var responseJSON any
	if json.Unmarshal(responseBody, &responseJSON) != nil {
		return nil, errors.New("MyAutoAPI 成功响应不是可解析的 JSON。")
	}
	images, err := extractImages(responseJSON, client)
	if err != nil {
		return nil, errors.New(redactError(err.Error(), apiKey))
	}
	if expectedCount > 0 && len(images) != expectedCount {
		return nil, fmt.Errorf("MyAutoAPI 返回了 %d 张图片，预期 %d 张。", len(images), expectedCount)
	}
	return images, nil
}

func decodeResponsesImageResponse(statusCode int, responseBody []byte, client httpDoer, apiKey string) ([][]byte, error) {
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("MyAutoAPI 返回 HTTP %d。", statusCode)
	}
	var responseJSON any
	if json.Unmarshal(responseBody, &responseJSON) != nil {
		return nil, errors.New("MyAutoAPI 成功响应不是可解析的 JSON。")
	}
	images, err := extractResponsesImages(responseJSON, client)
	if err != nil {
		return nil, errors.New(redactError(err.Error(), apiKey))
	}
	return images, nil
}

func outputSummary(outputFiles []string, requestedCount int) map[string]any {
	summary := map[string]any{
		"ok":              true,
		"requested_count": requestedCount,
		"output_count":    len(outputFiles),
	}
	if len(outputFiles) == 1 {
		summary["output_file"] = outputFiles[0]
		return summary
	}
	summary["output_files"] = outputFiles
	return summary
}

func codexHome() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("CODEX_HOME")); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法确定当前用户目录：%w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func loadAPIKey(root string) (string, error) {
	toolDir := filepath.Join(root, "tools", toolDirectoryName)
	source := keySourceDedicated
	sourcePath := filepath.Join(toolDir, keySourceFileName)
	if info, err := os.Lstat(sourcePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", fmt.Errorf("Key 来源文件不是普通文件：%s", sourcePath)
		}
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return "", fmt.Errorf("无法读取 Key 来源文件：%s", sourcePath)
		}
		source = strings.TrimSpace(string(content))
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("无法检查 Key 来源文件：%s", sourcePath)
	}
	if source == keySourceAuthJSON {
		return loadAuthJSONKey(root)
	}
	if source != keySourceDedicated {
		return "", fmt.Errorf("Key 来源设置无效：%s", sourcePath)
	}
	path := filepath.Join(toolDir, keyFileName)
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("无法读取专门画图分组 Key：%s", path)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("专门画图分组 Key 不是普通文件：%s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("无法读取专门画图分组 Key：%s", path)
	}
	key := strings.TrimSpace(string(content))
	if key == "" {
		return "", errors.New("专门画图分组 Key 文件为空。")
	}
	return key, nil
}

func loadAuthJSONKey(root string) (string, error) {
	path := filepath.Join(root, "auth.json")
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("无法读取 Codex auth.json：%s", path)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("Codex auth.json 不是普通文件：%s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("无法读取 Codex auth.json：%s", path)
	}
	var auth map[string]any
	if json.Unmarshal(content, &auth) != nil {
		return "", fmt.Errorf("Codex auth.json 格式无效：%s", path)
	}
	key, ok := auth["OPENAI_API_KEY"].(string)
	if !ok || strings.TrimSpace(key) == "" {
		return "", errors.New("Codex auth.json 中缺少可用的 OPENAI_API_KEY。")
	}
	return strings.TrimSpace(key), nil
}

func saveOutputImage(outputRoot, requestedName, extension string, content []byte) (string, error) {
	if err := os.MkdirAll(outputRoot, 0o777); err != nil {
		return "", err
	}
	baseName := sanitizeOutputName(requestedName)
	for attempt := 1; attempt <= 999; attempt++ {
		fileName := baseName + "." + extension
		if attempt > 1 {
			fileName = fmt.Sprintf("%s-%d.%s", baseName, attempt, extension)
		}
		path := filepath.Join(outputRoot, fileName)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(content)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(path)
			if writeErr != nil {
				return "", writeErr
			}
			return "", closeErr
		}
		return path, nil
	}
	return "", errors.New("同名图片过多，请换一个文件名。")
}

func saveOutputImages(outputRoot, requestedName string, contents [][]byte) ([]string, error) {
	paths := make([]string, 0, len(contents))
	for index, content := range contents {
		info := inspectImage(content)
		name := requestedName
		if len(contents) > 1 {
			base := sanitizeOutputName(requestedName)
			name = fmt.Sprintf("%s-%d", base, index+1)
		}
		path, err := saveOutputImage(outputRoot, name, info.extension, content)
		if err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func sanitizeOutputName(value string) string {
	name := strings.TrimSpace(value)
	if extension := filepath.Ext(name); isKnownImageExtension(extension) {
		name = strings.TrimSuffix(name, extension)
	}
	var builder strings.Builder
	lastWasSeparator := false
	for _, character := range name {
		invalid := unicode.IsControl(character) || strings.ContainsRune(`<>:"/\|?*`, character)
		if invalid {
			if !lastWasSeparator {
				builder.WriteRune('-')
				lastWasSeparator = true
			}
			continue
		}
		if unicode.IsSpace(character) {
			if builder.Len() > 0 && !lastWasSeparator {
				builder.WriteRune(' ')
				lastWasSeparator = true
			}
			continue
		}
		builder.WriteRune(character)
		lastWasSeparator = character == '-'
	}
	runes := []rune(strings.Trim(builder.String(), " .-"))
	if len(runes) > 60 {
		runes = runes[:60]
	}
	name = strings.Trim(string(runes), " .-")
	if name == "" {
		name = "生成图片"
	}
	if isWindowsReservedName(name) {
		name = "图片-" + name
	}
	return name
}

func isKnownImageExtension(value string) bool {
	switch strings.ToLower(value) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func isWindowsReservedName(value string) bool {
	base, _, _ := strings.Cut(value, ".")
	upper := strings.ToUpper(base)
	switch upper {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	for index := 1; index <= 9; index++ {
		if upper == fmt.Sprintf("COM%d", index) || upper == fmt.Sprintf("LPT%d", index) {
			return true
		}
	}
	return false
}

func readLimited(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxResponseBytes {
		return nil, errors.New("响应超过 100 MiB 限制")
	}
	return content, nil
}

func extractImages(responseJSON any, client httpDoer) ([][]byte, error) {
	root, ok := responseJSON.(map[string]any)
	if !ok {
		return nil, errors.New("图片接口返回的 JSON 结构无效。")
	}
	data, ok := root["data"].([]any)
	if !ok {
		return nil, errors.New("图片接口返回中没有 data 列表。")
	}
	images := make([][]byte, 0, len(data))
	var firstDownloadError error
	for _, rawItem := range data {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		if encoded, ok := item["b64_json"].(string); ok {
			if decoded := decodeImageBase64(encoded); decoded != nil {
				images = append(images, decoded)
				continue
			}
		}
		candidate, ok := item["url"].(string)
		if !ok || !isHTTPSURL(candidate) {
			continue
		}
		request, _ := http.NewRequest(http.MethodGet, candidate, nil)
		request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
		response, err := client.Do(request)
		if err != nil {
			if firstDownloadError == nil {
				firstDownloadError = fmt.Errorf("图片下载失败：%w", err)
			}
			continue
		}
		body, err := readLimited(response.Body)
		_ = response.Body.Close()
		if err != nil {
			if firstDownloadError == nil {
				firstDownloadError = fmt.Errorf("图片下载失败：%w", err)
			}
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			if firstDownloadError == nil {
				firstDownloadError = fmt.Errorf("图片 URL 下载返回 HTTP %d。", response.StatusCode)
			}
			continue
		}
		if len(body) == 0 {
			if firstDownloadError == nil {
				firstDownloadError = errors.New("图片 URL 下载结果为空。")
			}
			continue
		}
		if inspectImage(body).format == "unknown" {
			if firstDownloadError == nil {
				firstDownloadError = errors.New("图片 URL 返回的内容不是支持的图片格式。")
			}
			continue
		}
		images = append(images, body)
	}
	if len(images) == 0 {
		if firstDownloadError != nil {
			return nil, firstDownloadError
		}
		return nil, errors.New("图片接口未返回可保存的 b64_json 或 HTTPS URL。")
	}
	return images, nil
}

func extractResponsesImages(responseJSON any, client httpDoer) ([][]byte, error) {
	root, ok := responseJSON.(map[string]any)
	if !ok {
		return nil, errors.New("Responses 图片接口返回的 JSON 结构无效。")
	}
	output, ok := root["output"]
	if !ok {
		return nil, errors.New("Responses 图片接口返回中没有 output。")
	}
	images := make([][]byte, 0, 1)
	urls := make([]string, 0, 1)
	collectResponsesImageCandidates(output, &images, &urls)
	var firstDownloadError error
	for _, candidate := range urls {
		content, err := downloadImage(candidate, client)
		if err != nil {
			if firstDownloadError == nil {
				firstDownloadError = err
			}
			continue
		}
		appendUniqueImage(&images, content)
	}
	if len(images) == 0 {
		if firstDownloadError != nil {
			return nil, firstDownloadError
		}
		return nil, errors.New("Responses 图片接口未返回可保存的图片。")
	}
	return images, nil
}

func collectResponsesImageCandidates(value any, images *[][]byte, urls *[]string) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectResponsesImageCandidates(item, images, urls)
		}
	case map[string]any:
		for key, item := range typed {
			text, isString := item.(string)
			switch key {
			case "b64_json", "data", "image_base64":
				if isString {
					if decoded := decodeImageBase64(text); decoded != nil {
						appendUniqueImage(images, decoded)
					}
				}
			case "url", "image_url":
				if isString {
					collectImageString(text, images, urls)
				}
			}
			collectResponsesImageCandidates(item, images, urls)
		}
	case string:
		collectImageString(typed, images, urls)
	}
}

func collectImageString(value string, images *[][]byte, urls *[]string) {
	for _, candidate := range dataImagePattern.FindAllString(value, -1) {
		if decoded := decodeImageBase64(candidate); decoded != nil {
			appendUniqueImage(images, decoded)
		}
	}
	if isHTTPSURL(strings.TrimSpace(value)) {
		appendUniqueString(urls, strings.TrimSpace(value))
	}
	for _, match := range markdownImagePattern.FindAllStringSubmatch(value, -1) {
		if len(match) > 1 && isHTTPSURL(match[1]) {
			appendUniqueString(urls, match[1])
		}
	}
}

func appendUniqueImage(images *[][]byte, content []byte) {
	for _, existing := range *images {
		if bytes.Equal(existing, content) {
			return
		}
	}
	*images = append(*images, content)
}

func appendUniqueString(values *[]string, candidate string) {
	for _, existing := range *values {
		if existing == candidate {
			return
		}
	}
	*values = append(*values, candidate)
}

func downloadImage(candidate string, client httpDoer) ([]byte, error) {
	request, _ := http.NewRequest(http.MethodGet, candidate, nil)
	request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("图片下载失败：%w", err)
	}
	defer response.Body.Close()
	body, err := readLimited(response.Body)
	if err != nil {
		return nil, fmt.Errorf("图片下载失败：%w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("图片 URL 下载返回 HTTP %d。", response.StatusCode)
	}
	if len(body) == 0 || inspectImage(body).format == "unknown" {
		return nil, errors.New("图片 URL 返回的内容不是支持的图片格式。")
	}
	return body, nil
}

func decodeImageBase64(value string) []byte {
	candidate := strings.TrimSpace(value)
	if strings.HasPrefix(candidate, "data:") {
		if comma := strings.IndexByte(candidate, ','); comma >= 0 {
			candidate = candidate[comma+1:]
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(candidate)
	if err != nil || len(decoded) == 0 || inspectImage(decoded).format == "unknown" {
		return nil
	}
	return decoded
}

func isHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func inspectImage(content []byte) imageInfo {
	if len(content) >= 24 && bytes.Equal(content[:8], []byte("\x89PNG\r\n\x1a\n")) {
		return imageInfo{"png", "png", []int{int(binary.BigEndian.Uint32(content[16:20])), int(binary.BigEndian.Uint32(content[20:24]))}}
	}
	if dimensions := jpegDimensions(content); dimensions != nil {
		return imageInfo{"jpeg", "jpg", dimensions}
	}
	if len(content) >= 10 && (string(content[:6]) == "GIF87a" || string(content[:6]) == "GIF89a") {
		return imageInfo{"gif", "gif", []int{int(binary.LittleEndian.Uint16(content[6:8])), int(binary.LittleEndian.Uint16(content[8:10]))}}
	}
	if len(content) >= 12 && string(content[:4]) == "RIFF" && string(content[8:12]) == "WEBP" {
		var dimensions []int
		if len(content) >= 30 && string(content[12:16]) == "VP8X" {
			width := int(content[24]) | int(content[25])<<8 | int(content[26])<<16
			height := int(content[27]) | int(content[28])<<8 | int(content[29])<<16
			dimensions = []int{width + 1, height + 1}
		}
		return imageInfo{"webp", "webp", dimensions}
	}
	return imageInfo{"unknown", "bin", nil}
}

func jpegDimensions(content []byte) []int {
	if len(content) < 4 || content[0] != 0xff || content[1] != 0xd8 {
		return nil
	}
	for position := 2; position+4 <= len(content); {
		if content[position] != 0xff {
			position++
			continue
		}
		for position < len(content) && content[position] == 0xff {
			position++
		}
		if position >= len(content) {
			break
		}
		marker := content[position]
		position++
		if marker == 0xd8 || marker == 0xd9 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if position+2 > len(content) {
			break
		}
		length := int(binary.BigEndian.Uint16(content[position : position+2]))
		if length < 2 || position+length > len(content) {
			break
		}
		if isJPEGStartOfFrame(marker) && length >= 7 {
			height := int(binary.BigEndian.Uint16(content[position+3 : position+5]))
			width := int(binary.BigEndian.Uint16(content[position+5 : position+7]))
			return []int{width, height}
		}
		position += length
	}
	return nil
}

func isJPEGStartOfFrame(marker byte) bool {
	switch marker {
	case 0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
		return true
	default:
		return false
	}
}

func redactError(message, secret string) string {
	if secret == "" {
		return message
	}
	return strings.ReplaceAll(message, secret, "[REDACTED]")
}
