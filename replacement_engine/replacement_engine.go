package replacement_engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const string_replacement_prefix = "STRING_REPLACEMENT"

func ReplaceInString(src string, replacementValues map[string]string) (string, error) {
	if err := validateValues(replacementValues); err != nil {
		return "", err
	}
	ret := src
	for key, value := range replacementValues {
		ret = strings.Replace(ret, key, value, -1)
	}

	str_replacement_index := strings.Index(ret, string_replacement_prefix)

	if str_replacement_index > -1 {

		return "", fmt.Errorf("not all place holders replaced. Starting from %d:  %s", str_replacement_index, src[str_replacement_index:])
	}
	return ret, nil

}

func validateValues(src map[string]string) error {
	ret := []string{}
	for key, _ := range src {
		if !strings.HasPrefix(key, string_replacement_prefix) {
			ret = append(ret, fmt.Sprintf("Key '%s' has no prefix %s ", key, string_replacement_prefix))
		}

	}
	if len(ret) > 0 {
		return fmt.Errorf("input errors: %s", strings.Join(ret, "\n"))
	}
	return nil
}

func ReplaceInTemplateFiles(srcDir string, dstDir string, replacementValues map[string]string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "template_") {
			continue
		}
		_, err := ReplaceInTemplateFile(filepath.Join(srcDir, entry.Name()), dstDir, replacementValues)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReplaceInTemplateFile(srcFile string, dstDir string, replacementValues map[string]string) (string, error) {
	dstFileName := filepath.Base(srcFile)
	if !strings.HasPrefix(dstFileName, "template_") {
		return "", fmt.Errorf("expected 'template_' prefix in source file name: %s", srcFile)
	}
	dstFileName = dstFileName[len("template_"):]
	content, err := os.ReadFile(srcFile)
	if err != nil {
		fmt.Println("Error Reading file:", err)
		return "", err
	}
	contentReplaced, err := ReplaceInString(string(content), replacementValues)
	if err != nil {
		return "", err
	}

	byteContent := []byte(contentReplaced)
	dstFilePath := filepath.Join(dstDir, dstFileName)
	err = os.WriteFile(dstFilePath, byteContent, 0644)
	if err != nil {
		fmt.Println("Error Writing file:", err)
		return "", err
	}
	return dstFilePath, nil
}
