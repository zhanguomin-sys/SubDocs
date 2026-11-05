package connect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// jsonObject is a JSON object that keeps its keys in file order, so a file
// the assistant only patches keeps the order its owner gave it. Values stay
// as raw JSON and are never re-encoded unless the assistant sets them.
type jsonObject struct {
	keys   []string
	values map[string]json.RawMessage
}

func newJSONObject() *jsonObject {
	return &jsonObject{values: map[string]json.RawMessage{}}
}

// parseJSONObject reads a JSON object; a UTF-8 BOM is ignored and an empty
// file counts as an empty object.
func parseJSONObject(data []byte) (*jsonObject, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	object := newJSONObject()
	if len(bytes.TrimSpace(data)) == 0 {
		return object, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("不是 JSON 对象")
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, _ := token.(string)
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if _, seen := object.values[key]; !seen {
			object.keys = append(object.keys, key)
		}
		object.values[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err == nil {
		return nil, errors.New("JSON 对象后面还有多余内容")
	}
	return object, nil
}

func (o *jsonObject) get(key string) (json.RawMessage, bool) {
	value, ok := o.values[key]
	return value, ok
}

// getString returns a string value, or "" when missing or not a string.
func (o *jsonObject) getString(key string) string {
	var text string
	if value, ok := o.values[key]; ok && json.Unmarshal(value, &text) == nil {
		return text
	}
	return ""
}

// set replaces a value in place, or appends a new key at the end.
func (o *jsonObject) set(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = raw
	return nil
}

func (o *jsonObject) remove(key string) {
	if _, ok := o.values[key]; !ok {
		return
	}
	delete(o.values, key)
	for i, existing := range o.keys {
		if existing == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// encode writes the object with two-space indentation and a final newline.
func (o *jsonObject) encode() ([]byte, error) {
	var compact bytes.Buffer
	compact.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			compact.WriteByte(',')
		}
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		compact.Write(name)
		compact.WriteByte(':')
		compact.Write(o.values[key])
	}
	compact.WriteByte('}')
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// readJSONObject reads path; a missing file is an empty object and exists
// reports false.
func readJSONObject(path string) (object *jsonObject, exists bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return newJSONObject(), false, nil
	}
	if err != nil {
		return nil, false, err
	}
	object, err = parseJSONObject(data)
	if err != nil {
		return nil, true, fmt.Errorf("%s 不是有效的 JSON（%v）", filepath.Base(path), err)
	}
	return object, true, nil
}

// fileChange is one file of a group written together. A nil data removes
// the file. backup asks for a .bak.yyyyMMddHHmmss copy of the old file.
type fileChange struct {
	path   string
	data   []byte
	backup bool
}

// writeFileGroup writes every change or none: all new contents go to
// temporary files first, then backups are made, then the files are replaced
// one by one. If a replacement fails, the files already replaced are put
// back from their backups (or removed when they did not exist before).
func writeFileGroup(changes []fileChange, now time.Time) error {
	type staged struct {
		fileChange
		temp     string
		original []byte
		existed  bool
	}
	var steps []*staged
	defer func() {
		for _, step := range steps {
			if step.temp != "" {
				os.Remove(step.temp)
			}
		}
	}()
	for _, change := range changes {
		step := &staged{fileChange: change}
		original, err := os.ReadFile(change.path)
		switch {
		case err == nil:
			step.original, step.existed = original, true
		case errors.Is(err, fs.ErrNotExist):
		default:
			return err
		}
		if change.data != nil {
			directory := filepath.Dir(change.path)
			if err := os.MkdirAll(directory, 0o700); err != nil {
				return err
			}
			temp, err := writeTemporary(directory, "."+filepath.Base(change.path)+".tmp.", change.data)
			if err != nil {
				return err
			}
			step.temp = temp
		}
		steps = append(steps, step)
	}
	for _, step := range steps {
		if step.backup && step.existed {
			if _, err := backupIfExists(step.path, now); err != nil {
				return err
			}
		}
	}
	for i, step := range steps {
		var err error
		if step.data == nil {
			if step.existed {
				err = os.Remove(step.path)
			}
		} else {
			err = renameFile(step.temp, step.path)
			if err == nil {
				step.temp = ""
			}
		}
		if err != nil {
			for _, done := range steps[:i] {
				restoreContent(done.path, done.original, done.existed)
			}
			return err
		}
	}
	return nil
}

// restoreContent puts a file back to what it held before, after a failure.
func restoreContent(path string, original []byte, existed bool) {
	if !existed {
		_ = os.Remove(path)
		return
	}
	temporary, err := writeTemporary(filepath.Dir(path), ".rollback.", original)
	if err != nil {
		return
	}
	if renameFile(temporary, path) != nil {
		_ = os.Remove(temporary)
	}
}
