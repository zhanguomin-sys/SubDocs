package main

import (
	"bytes"
	"fmt"
	"image/gif"
	"os"
)

type gifOutputInfo struct {
	OutputFile       string `json:"output_file"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	FrameCount       int    `json:"frame_count"`
	FrameDurationsMS []int  `json:"frame_durations_ms"`
	DurationMS       int    `json:"duration_ms"`
	LoopCount        int    `json:"loop_count"`
}

func addGIFSummary(summary map[string]any, outputFiles []string, options generationOptions) {
	if options.outputFormat != "gif" {
		return
	}
	animations := make([]gifOutputInfo, 0, len(outputFiles))
	warnings := make([]string, 0)
	for _, path := range outputFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, "已保存结果，但无法读取实际 GIF 动画信息。")
			continue
		}
		if inspectImage(content).format != "gif" {
			warnings = append(warnings, "接口未返回 GIF，已按真实图片格式保存。")
			continue
		}
		info, err := inspectGIF(content)
		if err != nil {
			warnings = append(warnings, "已保留原始 GIF，但无法完整读取实际动画信息。")
			continue
		}
		info.OutputFile = path
		animations = append(animations, info)
		if options.frameCount > 0 && info.FrameCount != options.frameCount {
			warnings = append(warnings, fmt.Sprintf("GIF 实际为 %d 帧，与请求的 %d 帧不一致。", info.FrameCount, options.frameCount))
		}
		for _, delay := range info.FrameDurationsMS {
			if options.frameDurationMS > 0 && delay != options.frameDurationMS {
				warnings = append(warnings, fmt.Sprintf("GIF 实际每帧时长与请求的 %d 毫秒不完全一致，以 animations 为准。", options.frameDurationMS))
				break
			}
		}
	}
	if len(animations) > 0 {
		summary["animations"] = animations
	}
	if len(warnings) > 0 {
		summary["warnings"] = warnings
	}
}

func inspectGIF(content []byte) (gifOutputInfo, error) {
	var info gifOutputInfo
	// Scan metadata first, without expanding LZW pixels. This prevents untrusted
	// responses from allocating unbounded memory through image/gif.DecodeAll.
	if err := validateGIFDecodeBudget(content); err != nil {
		return info, err
	}
	animation, err := gif.DecodeAll(bytes.NewReader(content))
	if err != nil {
		return info, err
	}
	info.Width, info.Height = animation.Config.Width, animation.Config.Height
	info.FrameCount = len(animation.Image)
	info.LoopCount = animation.LoopCount
	info.FrameDurationsMS = make([]int, len(animation.Delay))
	for i, delay := range animation.Delay {
		info.FrameDurationsMS[i] = delay * 10
		info.DurationMS += delay * 10
	}
	return info, nil
}

func validateGIFDecodeBudget(content []byte) error {
	if len(content) < 13 || inspectImage(content).format != "gif" {
		return fmt.Errorf("GIF 文件头无效")
	}
	position := 13
	if content[10]&0x80 != 0 {
		position += 3 * (1 << ((content[10] & 7) + 1))
	}
	pixels := uint64(0)
	frames := 0
	skipBlocks := func() bool {
		for position < len(content) {
			size := int(content[position])
			position++
			if size == 0 {
				return true
			}
			if position+size > len(content) {
				return false
			}
			position += size
		}
		return false
	}
	for position < len(content) {
		switch content[position] {
		case 0x3b:
			if frames == 0 {
				return fmt.Errorf("GIF 不含图像帧")
			}
			return nil
		case 0x21:
			position += 2
			if !skipBlocks() {
				return fmt.Errorf("GIF 扩展块不完整")
			}
		case 0x2c:
			if position+10 > len(content) {
				return fmt.Errorf("GIF 图像描述不完整")
			}
			width := uint64(content[position+5]) | uint64(content[position+6])<<8
			height := uint64(content[position+7]) | uint64(content[position+8])<<8
			pixels += width * height
			frames++
			if pixels > uint64(maxImagePixels)*16 || frames > 256 {
				return fmt.Errorf("GIF 超过动画信息读取预算")
			}
			packed := content[position+9]
			position += 10
			if packed&0x80 != 0 {
				position += 3 * (1 << ((packed & 7) + 1))
			}
			position++ // LZW minimum code size.
			if !skipBlocks() {
				return fmt.Errorf("GIF 图像数据不完整")
			}
		default:
			return fmt.Errorf("GIF 数据块无效")
		}
	}
	return fmt.Errorf("GIF 结束标记缺失")
}
