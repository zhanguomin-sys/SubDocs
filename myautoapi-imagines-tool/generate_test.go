package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var tinyPNG = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/T6R2JQAAAABJRU5ErkJggg==")
var tinyJPEG = []byte{0xff, 0xd8, 0xff, 0xc0, 0x00, 0x11, 0x08, 0x00, 0x01, 0x00, 0x01, 0x03, 0x01, 0x11, 0x00, 0x02, 0x11, 0x00, 0x03, 0x11, 0x00, 0xff, 0xd9}

func TestNativeGenerationWritesOnlyTheFinalImage(t *testing.T) {
	root := prepareNativeAuth(t)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != modelName || payload["size"] != defaultSize || payload["quality"] != defaultQuality || payload["output_format"] != defaultOutputFormat || payload["n"] != float64(1) {
			t.Fatalf("unexpected payload: %v", payload)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	options.outputName = "白色背景红苹果"
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	summary := map[string]any{}
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary) != 4 || summary["ok"] != true || summary["requested_count"] != float64(1) || summary["output_count"] != float64(1) {
		t.Fatalf("unexpected summary: %v", summary)
	}
	outputFile := summary["output_file"].(string)
	if filepath.Base(outputFile) != "白色背景红苹果.png" {
		t.Fatalf("unexpected output name: %s", outputFile)
	}
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, tinyPNG) {
		t.Fatal("saved image differs from response")
	}
	entries, err := os.ReadDir(filepath.Join(root, "images"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].IsDir() {
		t.Fatalf("images directory contains extra records: %v", entries)
	}
}

func TestNativeGenerationUsesIndependentConcurrentRequestsForMultipleImages(t *testing.T) {
	root := prepareNativeAuth(t)
	var requests atomic.Int32
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["n"] != float64(1) {
			t.Fatalf("unexpected image count: %v", payload["n"])
		}
		current := inFlight.Add(1)
		for {
			maximum := maxInFlight.Load()
			if current <= maximum || maxInFlight.CompareAndSwap(maximum, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		inFlight.Add(-1)
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	options.n = 3
	options.outputName = "三张苹果"
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("request count = %d, want 3", requests.Load())
	}
	if maxInFlight.Load() < 2 {
		t.Fatalf("multi-image requests were not concurrent: max in flight = %d", maxInFlight.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	files, ok := summary["output_files"].([]any)
	if !ok || len(files) != 3 || summary["output_file"] != nil {
		t.Fatalf("unexpected multi-image summary: %v", summary)
	}
	if summary["requested_count"] != float64(3) || summary["output_count"] != float64(3) {
		t.Fatalf("unexpected image counts: %v", summary)
	}
	entries, err := os.ReadDir(filepath.Join(root, "images"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected three final images, got %d", len(entries))
	}
	for index, entry := range entries {
		expected := fmt.Sprintf("三张苹果-%d.png", index+1)
		if entry.Name() != expected || entry.IsDir() {
			t.Fatalf("unexpected multi-image output: %s", entry.Name())
		}
	}
}

func TestNativeGenerationCapsOverReturnedImages(t *testing.T) {
	root := prepareNativeAuth(t)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{
				map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)},
				map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyJPEG)},
			},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["requested_count"] != float64(1) || summary["output_count"] != float64(1) {
		t.Fatalf("unexpected capped summary: %v", summary)
	}
	if filepath.Ext(summary["output_file"].(string)) != ".png" {
		t.Fatalf("the first returned image was not kept: %v", summary)
	}
	entries, err := os.ReadDir(filepath.Join(root, "images"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one capped image, got %v, %d", err, len(entries))
	}
}

func TestNativeGenerationKeepsPartialSuccess(t *testing.T) {
	root := prepareNativeAuth(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestNumber := requests.Add(1)
		if requestNumber == 2 {
			response.WriteHeader(http.StatusBadGateway)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	options.n = 3
	options.outputName = "部分成功"
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("request count = %d, want 3", requests.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	files, ok := summary["output_files"].([]any)
	if !ok || len(files) != 2 || summary["requested_count"] != float64(3) || summary["output_count"] != float64(2) {
		t.Fatalf("unexpected partial-success summary: %v", summary)
	}
	entries, err := os.ReadDir(filepath.Join(root, "images"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected two preserved images, got %v, %d", err, len(entries))
	}
}

func TestGrokGenerationUsesDedicatedParametersAndActualImageFormat(t *testing.T) {
	root := prepareNativeAuth(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != grokModelName || payload["n"] != float64(1) || payload["resolution"] != "2k" || payload["aspect_ratio"] != "16:9" || payload["quality"] != "high" || payload["response_format"] != "b64_json" {
			t.Fatalf("unexpected Grok payload: %v", payload)
		}
		if _, ok := payload["size"]; ok {
			t.Fatalf("Grok payload contains size: %v", payload)
		}
		if _, ok := payload["output_format"]; ok {
			t.Fatalf("Grok payload contains output_format: %v", payload)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyJPEG)}},
		})
	}))
	defer server.Close()

	options, err := parseGenerationOptions([]string{
		"--prompt", "蓝色陶瓷杯",
		"--model", grokModelName,
		"--n", "3",
		"--resolution", "2K",
		"--aspect-ratio", "16:9",
		"--quality", "HIGH",
		"--output-dir", filepath.Join(root, "images"),
		"--output-name", "Grok 蓝色杯子",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("Grok request count = %d, want 3", requests.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	files, ok := summary["output_files"].([]any)
	if !ok || len(files) != 3 {
		t.Fatalf("unexpected Grok output: %v", summary)
	}
	for _, rawPath := range files {
		if filepath.Ext(rawPath.(string)) != ".jpg" {
			t.Fatalf("Grok JPEG used wrong extension: %s", rawPath)
		}
	}
}

func TestBananaGenerationUsesResponsesParametersAndNestedImage(t *testing.T) {
	root := prepareNativeAuth(t)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != bananaProModelName || payload["stream"] != false {
			t.Fatalf("unexpected Banana payload: %v", payload)
		}
		for _, unsupported := range []string{"prompt", "n", "size", "quality", "output_format"} {
			if _, ok := payload[unsupported]; ok {
				t.Fatalf("Banana payload contains %s: %v", unsupported, payload)
			}
		}
		input := payload["input"].([]any)
		message := input[0].(map[string]any)
		content := message["content"].([]any)
		if message["role"] != "user" || len(content) != 1 || content[0].(map[string]any)["type"] != "input_text" {
			t.Fatalf("unexpected Banana input: %v", input)
		}
		format := payload["response_format"].(map[string]any)
		if format["type"] != "image" || format["mime_type"] != "image/png" || format["aspect_ratio"] != "16:9" || format["image_size"] != "2K" {
			t.Fatalf("unexpected Banana response format: %v", format)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(responsesImageBody(tinyJPEG, "image/png"))
	}))
	defer server.Close()

	options, err := parseGenerationOptions([]string{
		"--prompt", "蓝色背景上的白色方块",
		"--model", bananaProModelName,
		"--resolution", "2K",
		"--aspect-ratio", "16:9",
		"--output-dir", filepath.Join(root, "images"),
		"--output-name", "Banana 结果",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(summary["output_file"].(string)) != ".jpg" {
		t.Fatalf("Banana result did not use actual JPEG signature: %v", summary)
	}
}

func TestBananaMultiGenerationUsesOneRequestPerOutput(t *testing.T) {
	root := prepareNativeAuth(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if _, ok := payload["n"]; ok {
			t.Fatalf("Responses request must not contain n: %v", payload)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(responsesImageBody(tinyPNG, "image/png"))
	}))
	defer server.Close()

	options, err := parseGenerationOptions([]string{
		"--prompt", "三张测试图",
		"--model", banana2ModelName,
		"--n", "3",
		"--output-dir", filepath.Join(root, "images"),
		"--output-name", "Banana 三张图",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("Banana request count = %d, want 3", requests.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if files, ok := summary["output_files"].([]any); !ok || len(files) != 3 {
		t.Fatalf("unexpected Banana multi-image summary: %v", summary)
	}
}

func TestSingleRequestModelsUseOneRequestPerOutput(t *testing.T) {
	models := []struct {
		name    string
		quality string
	}{
		{thModelName, thDefaultQuality},
		{adobeModelName, adobeDefaultQuality},
	}
	for _, model := range models {
		t.Run(model.name, func(t *testing.T) {
			root := prepareNativeAuth(t)
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload["model"] != model.name || payload["n"] != float64(1) || payload["quality"] != model.quality {
					t.Fatalf("unexpected single-image request: %v", payload)
				}
				response.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(response).Encode(map[string]any{
					"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
				})
			}))
			defer server.Close()

			options, err := parseGenerationOptions([]string{
				"--prompt", "一只蓝色陶瓷杯",
				"--model", model.name,
				"--n", "3",
				"--output-dir", filepath.Join(root, "images"),
				"--output-name", "蓝色陶瓷杯",
			})
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 3 {
				t.Fatalf("single-image request count = %d, want 3", requests.Load())
			}
			var summary map[string]any
			if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
				t.Fatal(err)
			}
			if files, ok := summary["output_files"].([]any); !ok || len(files) != 3 {
				t.Fatalf("unexpected multi-image summary: %v", summary)
			}
		})
	}
}

func TestNativeEditSendsMultipartImages(t *testing.T) {
	root := prepareNativeAuth(t)
	inputOne := filepath.Join(root, "参考图.png")
	inputTwo := filepath.Join(root, "主体图.png")
	if err := os.WriteFile(inputOne, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputTwo, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("unexpected edit request: %s %s", request.Method, request.Header.Get("Content-Type"))
		}
		if err := request.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if request.FormValue("model") != modelName || request.FormValue("prompt") != "只修改背景，保留主体" || request.FormValue("n") != "1" {
			t.Fatalf("unexpected edit fields: %v", request.Form)
		}
		files := request.MultipartForm.File["image[]"]
		if len(files) != 2 {
			t.Fatalf("expected two input images, got %d", len(files))
		}
		for _, header := range files {
			file, err := header.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, readErr := io.ReadAll(file)
			_ = file.Close()
			if readErr != nil || !bytes.Equal(content, tinyPNG) {
				t.Fatalf("multipart image content was changed")
			}
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	options.images = []string{inputOne, inputTwo}
	options.prompt = "只修改背景，保留主体"
	options.outputName = "背景修改结果"
	var output bytes.Buffer
	if err := runEdit(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(summary["output_file"].(string)) != "背景修改结果.png" {
		t.Fatalf("unexpected edit output: %v", summary)
	}
}

func TestGrokEditUsesVerifiedMultipartFields(t *testing.T) {
	root := prepareNativeAuth(t)
	input := filepath.Join(root, "参考图.png")
	if err := os.WriteFile(input, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := request.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if request.FormValue("model") != grokModelName || request.FormValue("n") != "1" || request.FormValue("response_format") != "b64_json" {
			t.Fatalf("unexpected Grok edit form: %v", request.MultipartForm.Value)
		}
		for _, field := range []string{"size", "quality", "output_format", "aspect_ratio", "resolution"} {
			if _, ok := request.MultipartForm.Value[field]; ok {
				t.Fatalf("Grok edit contains unverified field %s: %v", field, request.MultipartForm.Value)
			}
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyJPEG)}},
		})
	}))
	defer server.Close()

	options, err := parseEditOptions([]string{
		"--prompt", "把杯子改成红色",
		"--model", grokModelName,
		"--image", input,
		"--output-dir", filepath.Join(root, "images"),
		"--output-name", "红色杯子",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runEdit(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(summary["output_file"].(string)) != ".jpg" {
		t.Fatalf("unexpected Grok edit output: %v", summary)
	}
}

func TestBananaEditEmbedsReferenceImageInResponsesInput(t *testing.T) {
	root := prepareNativeAuth(t)
	input := filepath.Join(root, "参考图.png")
	if err := os.WriteFile(input, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected Banana edit content type: %s", request.Header.Get("Content-Type"))
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		content := payload["input"].([]any)[0].(map[string]any)["content"].([]any)
		if len(content) != 2 || content[0].(map[string]any)["type"] != "input_text" {
			t.Fatalf("unexpected Banana edit input: %v", content)
		}
		image := content[1].(map[string]any)
		imageURL, _ := image["image_url"].(string)
		if image["type"] != "input_image" || !strings.HasPrefix(imageURL, "data:image/png;base64,") {
			t.Fatalf("reference image was not embedded correctly: %v", image)
		}
		decoded := decodeImageBase64(imageURL)
		if !bytes.Equal(decoded, tinyPNG) {
			t.Fatal("embedded reference image differs from the input")
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(responsesImageBody(tinyPNG, "image/png"))
	}))
	defer server.Close()

	options, err := parseEditOptions([]string{
		"--prompt", "把背景改成红色",
		"--model", bananaProModelName,
		"--image", input,
		"--resolution", "4K",
		"--aspect-ratio", "4:3",
		"--output-dir", filepath.Join(root, "images"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runEdit(options, &bytes.Buffer{}, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
}

func TestBanana2EditEmbedsTwoAndFourReferenceImages(t *testing.T) {
	root := prepareNativeAuth(t)
	paths := []string{
		filepath.Join(root, "参考图-1.png"),
		filepath.Join(root, "参考图-2.jpg"),
		filepath.Join(root, "参考图-3.png"),
		filepath.Join(root, "参考图-4.jpg"),
	}
	for index, path := range paths {
		content := tinyPNG
		if index%2 == 1 {
			content = tinyJPEG
		}
		if err := os.WriteFile(path, content, 0o666); err != nil {
			t.Fatal(err)
		}
	}
	var requestIndex atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		index := int(requestIndex.Add(1)) - 1
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != banana2ModelName {
			t.Fatalf("unexpected Banana 2 model: %v", payload)
		}
		for _, unsupported := range []string{"prompt", "n", "size", "quality", "output_format"} {
			if _, ok := payload[unsupported]; ok {
				t.Fatalf("Banana 2 payload contains %s: %v", unsupported, payload)
			}
		}
		content := payload["input"].([]any)[0].(map[string]any)["content"].([]any)
		wantImages := 2
		if index == 1 {
			wantImages = 4
		}
		if len(content) != wantImages+1 {
			t.Fatalf("reference image count = %d, want %d: %v", len(content)-1, wantImages, content)
		}
		for imageIndex, raw := range content[1:] {
			image := raw.(map[string]any)
			imageURL := image["image_url"].(string)
			wantMime := "image/png"
			if imageIndex%2 == 1 {
				wantMime = "image/jpeg"
			}
			if image["type"] != "input_image" || !strings.HasPrefix(imageURL, "data:"+wantMime+";base64,") {
				t.Fatalf("reference image %d was not embedded correctly: %v", imageIndex, image)
			}
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(responsesImageBody(tinyPNG, "image/png"))
	}))
	defer server.Close()

	for _, count := range []int{2, 4} {
		args := []string{
			"--prompt", "保留主体并修改背景",
			"--model", banana2ModelName,
			"--resolution", "2k",
			"--aspect-ratio", "16:9",
			"--output-dir", filepath.Join(root, "images"),
		}
		for _, path := range paths[:count] {
			args = append(args, "--image", path)
		}
		options, err := parseEditOptions(args)
		if err != nil {
			t.Fatal(err)
		}
		if err := runEdit(options, &bytes.Buffer{}, server.Client(), server.URL); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBananaEmptyReferenceReportsActualSizeRule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "空图片.png")
	if err := os.WriteFile(path, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	_, err := encodeBananaInputImage(path)
	if err == nil || !strings.Contains(err.Error(), "大小必须大于 0 且不超过 7 MiB") {
		t.Fatalf("unexpected empty reference error: %v", err)
	}
}

func TestTHModelEditUsesSelectedModel(t *testing.T) {
	root := prepareNativeAuth(t)
	input := filepath.Join(root, "参考图.png")
	if err := os.WriteFile(input, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := request.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if request.FormValue("model") != thModelName || request.FormValue("quality") != thDefaultQuality || request.FormValue("size") != thDefaultSize {
			t.Fatalf("unexpected TH edit fields: %v", request.Form)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options, err := parseEditOptions([]string{
		"--prompt", "把杯子改成红色",
		"--model", thModelName,
		"--image", input,
		"--output-dir", filepath.Join(root, "images"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runEdit(options, &bytes.Buffer{}, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
}

func TestNativeEditRejectsNonImageBeforeRequest(t *testing.T) {
	root := prepareNativeAuth(t)
	input := filepath.Join(root, "不是图片.txt")
	if err := os.WriteFile(input, []byte("not an image"), 0o666); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	options := testGenerationOptions(filepath.Join(root, "images"))
	options.images = []string{input}
	if err := runEdit(options, &bytes.Buffer{}, server.Client(), server.URL); err == nil || !strings.Contains(err.Error(), "不是支持的图片格式") {
		t.Fatalf("expected local non-image rejection, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatal("non-image input reached the API")
	}
	if _, err := os.Stat(filepath.Join(root, "images")); !os.IsNotExist(err) {
		t.Fatalf("non-image edit created user output: %v", err)
	}
}

func TestNativeEditCompletesThreeImagesWhenUpstreamIgnoresN(t *testing.T) {
	root := prepareNativeAuth(t)
	input := filepath.Join(root, "参考图.png")
	if err := os.WriteFile(input, tinyPNG, 0o666); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if err := request.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if request.FormValue("n") != "1" {
			t.Fatalf("independent edit request used n=%s", request.FormValue("n"))
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
	defer server.Close()

	options := testGenerationOptions(filepath.Join(root, "images"))
	options.images = []string{input}
	options.n = 3
	options.outputName = "三张参考图结果"
	var output bytes.Buffer
	if err := runEdit(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("expected three independent edit requests, got %d", requests.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if files, ok := summary["output_files"].([]any); !ok || len(files) != 3 {
		t.Fatalf("expected three edit output files, got %v", summary)
	}
}

func TestNativeGenerationCompletesMultiImageWhenUpstreamIgnoresN(t *testing.T) {
	root := prepareNativeAuth(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{
				map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)},
			},
		})
	}))
	defer server.Close()
	options := testGenerationOptions(filepath.Join(root, "images"))
	options.n = 3
	var output bytes.Buffer
	if err := runGeneration(options, &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("expected three independent generation requests, got %d", requests.Load())
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if files, ok := summary["output_files"].([]any); !ok || len(files) != 3 {
		t.Fatalf("expected three output files, got %v", summary)
	}
	entries, err := os.ReadDir(filepath.Join(root, "images"))
	if err != nil || len(entries) != 3 {
		t.Fatalf("expected three images in output directory, got %v, %d", err, len(entries))
	}
}

func TestNativeGenerationUsesDedicatedKeyWithoutAuthJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	toolDir := filepath.Join(root, "tools", toolDirectoryName)
	if err := os.MkdirAll(toolDir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, keyFileName), []byte("dedicated-only-key"), 0o666); err != nil {
		t.Fatal(err)
	}
	server := newImageServer(t)
	defer server.Close()
	if err := runGeneration(testGenerationOptions(filepath.Join(root, "images")), &bytes.Buffer{}, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
}

func TestNativeGenerationHiddenAuthJSONModeRemainsAvailable(t *testing.T) {
	root := prepareNativeAuth(t)
	toolDir := filepath.Join(root, "tools", toolDirectoryName)
	if err := os.WriteFile(filepath.Join(toolDir, keySourceFileName), []byte(keySourceAuthJSON), 0o666); err != nil {
		t.Fatal(err)
	}
	server := newImageServer(t)
	defer server.Close()
	if err := runGeneration(testGenerationOptions(filepath.Join(root, "images")), &bytes.Buffer{}, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
}

func TestNativeGenerationFallsBackToHTTPSURLOnce(t *testing.T) {
	root := prepareNativeAuth(t)
	var imageCalls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/image.png" {
			imageCalls.Add(1)
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write(tinyPNG)
			return
		}
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{
				"b64_json": "invalid",
				"url":      "https://" + request.Host + "/image.png",
			}},
		})
	}))
	defer server.Close()

	var output bytes.Buffer
	if err := runGeneration(testGenerationOptions(filepath.Join(root, "images")), &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if imageCalls.Load() != 1 {
		t.Fatalf("image URL requested %d times", imageCalls.Load())
	}
	summary := map[string]any{}
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(summary["output_file"].(string)); err != nil {
		t.Fatal(err)
	}
}

func TestNativeGenerationFailureLeavesNoUserFiles(t *testing.T) {
	root := prepareNativeAuth(t)
	outputDir := filepath.Join(root, "images")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(response, `{"error":"fake-key-not-real"}`)
	}))
	defer server.Close()

	err := runGeneration(testGenerationOptions(outputDir), &bytes.Buffer{}, server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") || strings.Contains(err.Error(), "fake-key-not-real") {
		t.Fatalf("unexpected failure: %v", err)
	}
	if _, statErr := os.Stat(outputDir); !os.IsNotExist(statErr) {
		t.Fatalf("failure created user output: %v", statErr)
	}
}

func TestNativeCLIReturnsSimpleGenerationFailureAndDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"generate", "--prompt", "测试", "--n", "0"}, &stdout, &stderr)
	if exitCode != 1 || stdout.Len() != 0 {
		t.Fatalf("unexpected CLI result: exit=%d stdout=%q", exitCode, stdout.String())
	}
	var summary map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["ok"] != false || summary["error"] != generationFailureMessage {
		t.Fatalf("unexpected customer error: %v", summary)
	}
	diagnostic, _ := summary["diagnostic"].(string)
	if diagnostic == "" || !strings.Contains(diagnostic, "图片数量") {
		t.Fatalf("missing repair diagnostic: %v", summary)
	}
}

func TestNativeGenerationRejectsNonImageURLBody(t *testing.T) {
	root := prepareNativeAuth(t)
	outputDir := filepath.Join(root, "images")
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/not-image" {
			response.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(response, "upstream error page")
			return
		}
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"url": "https://" + request.Host + "/not-image"}},
		})
	}))
	defer server.Close()

	err := runGeneration(testGenerationOptions(outputDir), &bytes.Buffer{}, server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "不是支持的图片格式") {
		t.Fatalf("expected non-image rejection, got %v", err)
	}
	if _, statErr := os.Stat(outputDir); !os.IsNotExist(statErr) {
		t.Fatalf("non-image response created user output: %v", statErr)
	}
}

func TestNativeGenerationDoesNotRedirectAuthorizationToHTTP(t *testing.T) {
	root := prepareNativeAuth(t)
	var httpCalls atomic.Int32
	var receivedAuthorization atomic.Bool
	httpTarget := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		httpCalls.Add(1)
		if request.Header.Get("Authorization") != "" {
			receivedAuthorization.Store(true)
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer httpTarget.Close()

	redirectSource := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, httpTarget.URL, http.StatusTemporaryRedirect)
	}))
	defer redirectSource.Close()
	client := redirectSource.Client()
	client.CheckRedirect = secureRedirectPolicy

	err := runGeneration(testGenerationOptions(filepath.Join(root, "images")), &bytes.Buffer{}, client, redirectSource.URL)
	if err == nil {
		t.Fatal("expected API redirect to be rejected")
	}
	if httpCalls.Load() != 0 || receivedAuthorization.Load() {
		t.Fatal("Authorization request reached the HTTP redirect target")
	}
}

func TestNativeCLIHasNoKeyOrEndpointOptions(t *testing.T) {
	for _, arguments := range [][]string{
		{"--prompt", "x", "--api-key", "secret"},
		{"--prompt", "x", "--base-url", "https://example.com"},
		{"--prompt", "x", "--image", "input.png"},
	} {
		if _, err := parseGenerationOptions(arguments); err == nil {
			t.Fatalf("unsafe option was accepted: %v", arguments)
		}
	}
}

func TestNativeCLIDefaultsToFlareAt2KHigh(t *testing.T) {
	options, err := parseGenerationOptions([]string{"--prompt", "x"})
	if err != nil {
		t.Fatal(err)
	}
	if options.model != "gpt-image-2.5-flare" || options.size != "2048x2048" || options.quality != "high" || options.outputFormat != "png" {
		t.Fatalf("unexpected default options: %+v", options)
	}

	payload := generationPayload(options)
	if payload["model"] != "gpt-image-2.5-flare" || payload["size"] != "2048x2048" || payload["quality"] != "high" || payload["output_format"] != "png" {
		t.Fatalf("unexpected default payload: %v", payload)
	}
}

func TestNativeCLISupportsGPTImage25GenerationAndEdit(t *testing.T) {
	for _, model := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, quality := range []string{"low", "medium", "high", "xhigh", "max", "auto"} {
			options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", model, "--quality", quality})
			if err != nil {
				t.Fatalf("model %s rejected quality %s: %v", model, quality, err)
			}
			if options.size != "2048x2048" || options.quality != quality {
				t.Fatalf("unexpected options for model %s quality %s: %+v", model, quality, options)
			}
		}

		options, err := parseEditOptions([]string{"--prompt", "x", "--model", model, "--image", "input.png"})
		if err != nil {
			t.Fatalf("model %s rejected edit: %v", model, err)
		}
		fields := editFields(options)
		if fields["model"] != model || fields["size"] != "2048x2048" || fields["quality"] != "high" || fields["output_format"] != "png" {
			t.Fatalf("unexpected edit fields for model %s: %v", model, fields)
		}
	}
}

func TestNativeCLIKeepsGPTImage2DefaultsAndQualityBoundary(t *testing.T) {
	options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", "gpt-image-2"})
	if err != nil {
		t.Fatal(err)
	}
	if options.size != "1024x1024" || options.quality != "high" {
		t.Fatalf("gpt-image-2 defaults changed: %+v", options)
	}
	for _, quality := range []string{"xhigh", "max"} {
		if _, err := parseGenerationOptions([]string{"--prompt", "x", "--model", "gpt-image-2", "--quality", quality}); err == nil {
			t.Fatalf("gpt-image-2 accepted GPT Image 2.5-only quality %s", quality)
		}
	}
}

func TestNativeCLIValidatesImageParameters(t *testing.T) {
	if newHTTPClient().Timeout != 10*time.Minute {
		t.Fatalf("generation timeout = %s", newHTTPClient().Timeout)
	}

	validSizes := []string{
		"auto",
		"1024x1024",
		"1536x1024",
		"1024x1536",
		"2048x2048",
		"2048x1152",
		"1152x2048",
		"3840x2160",
		"2160x3840",
		"768x1024",
		"1024x768",
		"768x960",
		"960x768",
		"720x1280",
		"1280x720",
		"1344x576",
		"576x1344",
		"1536x512",
		"512x1536",
	}
	for _, size := range validSizes {
		arguments := []string{"--prompt", "x", "--size", size, "--quality", "AUTO", "--output-format", "PNG"}
		options, err := parseGenerationOptions(arguments)
		if err != nil {
			t.Fatalf("valid size %s rejected: %v", size, err)
		}
		if options.quality != "auto" || options.outputFormat != "png" {
			t.Fatalf("parameters were not normalized: %+v", options)
		}
	}
	multiOptions, err := parseGenerationOptions([]string{"--prompt", "x", "--n", "3"})
	if err != nil || multiOptions.n != 3 {
		t.Fatalf("three-image generation was not accepted: %+v, %v", multiOptions, err)
	}
	if _, err := parseEditOptions([]string{"--prompt", "x"}); err == nil {
		t.Fatal("edit accepted without an input image")
	}
	if _, err := parseEditOptions([]string{"--prompt", "x", "--image", "input.png", "--n", "11"}); err == nil {
		t.Fatal("edit accepted more than ten outputs")
	}

	invalidArguments := [][]string{
		{"--prompt", "x", "--size", "1025x1024"},
		{"--prompt", "x", "--size", "512x512"},
		{"--prompt", "x", "--size", "3840x3840"},
		{"--prompt", "x", "--size", "3088x1024"},
		{"--prompt", "x", "--size", "2560x640"},
		{"--prompt", "x", "--size", "640x2560"},
		{"--prompt", "x", "--size", "2560x320"},
		{"--prompt", "x", "--size", "320x2560"},
		{"--prompt", "x", "--size", "3:4"},
		{"--prompt", "x", "--aspect-ratio", "3:4"},
		{"--prompt", "x", "--quality", "ultra"},
		{"--prompt", "x", "--output-format", "jpeg"},
		{"--prompt", "x", "--n", "0"},
		{"--prompt", "x", "--n", "11"},
	}
	for _, arguments := range invalidArguments {
		if _, err := parseGenerationOptions(arguments); err == nil {
			t.Fatalf("invalid parameters were accepted: %v", arguments)
		}
	}
	if _, err := parseGenerationOptions([]string{"--prompt", "x", "--size", "3:4"}); err == nil || !strings.Contains(err.Error(), "WIDTHxHEIGHT") {
		t.Fatalf("ratio text was not rejected as a pixel size: %v", err)
	}
	if _, err := parseGenerationOptions([]string{"--prompt", "x", "--aspect-ratio", "3:4"}); err == nil || !strings.Contains(err.Error(), "--size") {
		t.Fatalf("GPT Image 2 accepted Banana's aspect-ratio form: %v", err)
	}
}

func TestNativeCLIUsesPerModelCapabilities(t *testing.T) {
	tests := []struct {
		model      string
		size       string
		quality    string
		ratio      string
		resolution string
	}{
		{modelName, defaultSize, defaultQuality, "", ""},
		{thModelName, thDefaultSize, thDefaultQuality, "", ""},
		{adobeModelName, adobeDefaultSize, adobeDefaultQuality, "", ""},
		{grokModelName, "", "", grokDefaultRatio, grokDefaultResolution},
		{banana2ModelName, "", "", bananaDefaultRatio, bananaDefaultResolution},
		{bananaProModelName, "", "", bananaDefaultRatio, bananaDefaultResolution},
	}
	for _, test := range tests {
		options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", test.model})
		if err != nil {
			t.Fatalf("supported model %s rejected: %v", test.model, err)
		}
		if options.size != test.size || options.quality != test.quality || options.aspectRatio != test.ratio || options.resolution != test.resolution {
			t.Fatalf("unexpected defaults for model %s: %+v", test.model, options)
		}
	}
	for _, size := range []string{"1024x1536", "1536x2304", "2304x3456"} {
		options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", adobeModelName, "--size", size})
		if err != nil {
			t.Fatalf("Adobe size %s rejected: %v", size, err)
		}
		if options.size != size || options.quality != adobeDefaultQuality {
			t.Fatalf("unexpected Adobe options for %s: %+v", size, options)
		}
	}
	for _, ratio := range []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"} {
		options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", grokModelName, "--aspect-ratio", ratio, "--resolution", "2K", "--quality", "medium"})
		if err != nil || options.aspectRatio != ratio || options.resolution != "2k" || options.quality != "medium" {
			t.Fatalf("Grok parameters rejected: %+v, %v", options, err)
		}
	}
	if options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", grokModelName, "--n", "9"}); err != nil || options.n != 9 {
		t.Fatalf("Grok nine-image request rejected: %+v, %v", options, err)
	}
	for _, resolution := range []string{"512", "1K", "2K", "4K"} {
		options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", banana2ModelName, "--resolution", resolution, "--aspect-ratio", "9:21"})
		if err != nil || options.resolution != strings.ToLower(resolution) || options.aspectRatio != "9:21" {
			t.Fatalf("Nano Banana 2 parameters rejected: %+v, %v", options, err)
		}
	}
	for _, resolution := range []string{"1K", "2K", "4K"} {
		options, err := parseGenerationOptions([]string{"--prompt", "x", "--model", bananaProModelName, "--resolution", resolution, "--aspect-ratio", "1:8"})
		if err != nil || options.resolution != strings.ToLower(resolution) || options.aspectRatio != "1:8" {
			t.Fatalf("Nano Banana Pro parameters rejected: %+v, %v", options, err)
		}
	}

	for _, arguments := range [][]string{
		{"--prompt", "x", "--model", "ad-gpt-image-2-mx"},
		{"--prompt", "x", "--model", "gpt-image-1k-adobe"},
		{"--prompt", "x", "--model", adobeModelName, "--quality", "high"},
		{"--prompt", "x", "--model", adobeModelName, "--size", "1024x1024"},
		{"--prompt", "x", "--model", adobeModelName, "--size", "2048x3072"},
		{"--prompt", "x", "--model", thModelName, "--quality", "medium"},
		{"--prompt", "x", "--model", thModelName, "--size", "1024x1536"},
		{"--prompt", "x", "--model", grokModelName, "--size", "1024x1024"},
		{"--prompt", "x", "--model", grokModelName, "--aspect-ratio", "21:9"},
		{"--prompt", "x", "--model", grokModelName, "--resolution", "4k"},
		{"--prompt", "x", "--model", grokModelName, "--n", "10"},
		{"--prompt", "x", "--model", modelName, "--aspect-ratio", "16:9"},
		{"--prompt", "x", "--model", banana2ModelName, "--size", "1024x1024"},
		{"--prompt", "x", "--model", banana2ModelName, "--quality", "high"},
		{"--prompt", "x", "--model", bananaProModelName, "--resolution", "512"},
		{"--prompt", "x", "--model", bananaProModelName, "--aspect-ratio", "7:5"},
	} {
		if _, err := parseGenerationOptions(arguments); err == nil {
			t.Fatalf("unsupported model parameters were accepted: %v", arguments)
		}
	}
	if _, err := parseEditOptions([]string{"--prompt", "x", "--model", grokModelName, "--image", "input.png", "--resolution", "2k"}); err == nil {
		t.Fatal("Grok edit accepted generation-only resolution")
	}
	for _, model := range []string{adobeModelName} {
		if _, err := parseEditOptions([]string{"--prompt", "x", "--model", model, "--image", "input.png"}); err == nil || !strings.Contains(err.Error(), "只支持文生图") {
			t.Fatalf("model %s edit should be rejected, got %v", model, err)
		}
	}
	for _, quality := range []string{"low", "high"} {
		options, err := parseEditOptions([]string{"--prompt", "x", "--model", thModelName, "--image", "input.png", "--quality", quality})
		if err != nil {
			t.Fatalf("TH edit quality %s rejected: %v", quality, err)
		}
		if options.size != thDefaultSize || options.model != thModelName {
			t.Fatalf("unexpected TH edit options: %+v", options)
		}
	}
}

func TestOutputNamesAreFlatSafeAndNeverOverwrite(t *testing.T) {
	outputDir := t.TempDir()
	first, err := saveOutputImage(outputDir, `  红苹果:白色/背景.png  `, "png", tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	second, err := saveOutputImage(outputDir, `红苹果:白色/背景.png`, "png", tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := saveOutputImage(outputDir, "CON", "png", tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	reservedWithSuffix, err := saveOutputImage(outputDir, "AUX.定稿", "png", tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	dotted, err := saveOutputImage(outputDir, "角色V1.2定稿", "png", tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "红苹果-白色-背景.png" || filepath.Base(second) != "红苹果-白色-背景-2.png" || filepath.Base(reserved) != "图片-CON.png" || filepath.Base(reservedWithSuffix) != "图片-AUX.定稿.png" || filepath.Base(dotted) != "角色V1.2定稿.png" {
		t.Fatalf("unexpected output paths: %s, %s, %s, %s, %s", first, second, reserved, reservedWithSuffix, dotted)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("unexpected output count: %d", len(entries))
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("output created a nested directory: %s", entry.Name())
		}
	}
}

func TestNativeGenerationDefaultsToWorkingDirectoryImages(t *testing.T) {
	root := prepareNativeAuth(t)
	workspace := t.TempDir()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDirectory) })

	server := newImageServer(t)
	defer server.Close()
	var output bytes.Buffer
	if err := runGeneration(testGenerationOptions(""), &output, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	summary := map[string]any{}
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	outputFile := summary["output_file"].(string)
	if filepath.Dir(outputFile) != filepath.Join(workspace, "images") {
		t.Fatalf("default output root = %s", filepath.Dir(outputFile))
	}
	if strings.HasPrefix(outputFile, root) {
		t.Fatalf("default output still uses CODEX_HOME: %s", outputFile)
	}
}

func newImageServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}},
		})
	}))
}

func responsesImageBody(content []byte, declaredMimeType string) map[string]any {
	dataURL := "data:" + declaredMimeType + ";base64," + base64.StdEncoding.EncodeToString(content)
	return map[string]any{
		"object": "response",
		"status": "completed",
		"output": []any{map[string]any{
			"type": "message",
			"content": []any{map[string]any{
				"type": "output_text",
				"text": "![image](" + dataURL + ")",
			}},
		}},
	}
}

func testGenerationOptions(outputDir string) generationOptions {
	return generationOptions{
		prompt:       "一颗红苹果",
		model:        modelName,
		n:            1,
		size:         defaultSize,
		quality:      defaultQuality,
		outputFormat: defaultOutputFormat,
		outputDir:    outputDir,
		outputName:   "红苹果",
	}
}

func prepareNativeAuth(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	content := []byte(`{"OPENAI_API_KEY":"fake-key-not-real"}`)
	if err := os.WriteFile(filepath.Join(root, "auth.json"), content, 0o666); err != nil {
		t.Fatal(err)
	}
	toolDir := filepath.Join(root, "tools", toolDirectoryName)
	if err := os.MkdirAll(toolDir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, keyFileName), []byte("fake-key-not-real"), 0o666); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustDecodeBase64(value string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return decoded
}
