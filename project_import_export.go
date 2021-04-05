//
// Copyright 2021, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ProjectImportExportService handles communication with the project
// import/export related methods of the GitLab API.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/user/project/settings/import_export.html
type ProjectImportExportService struct {
	client *Client
}

// ImportStatus represents a project import status.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#import-status
type ImportStatus struct {
	ID                int        `json:"id"`
	Description       string     `json:"description"`
	Name              string     `json:"name"`
	NameWithNamespace string     `json:"name_with_namespace"`
	Path              string     `json:"path"`
	PathWithNamespace string     `json:"path_with_namespace"`
	CreateAt          *time.Time `json:"create_at"`
	ImportStatus      string     `json:"import_status"`
}

func (s ImportStatus) String() string {
	return Stringify(s)
}

// ExportStatus represents a project export status.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#export-status
type ExportStatus struct {
	ID                int        `json:"id"`
	Description       string     `json:"description"`
	Name              string     `json:"name"`
	NameWithNamespace string     `json:"name_with_namespace"`
	Path              string     `json:"path"`
	PathWithNamespace string     `json:"path_with_namespace"`
	CreatedAt         *time.Time `json:"created_at"`
	ExportStatus      string     `json:"export_status"`
	Message           string     `json:"message"`
	Links             struct {
		APIURL string `json:"api_url"`
		WebURL string `json:"web_url"`
	} `json:"_links"`
}

func (s ExportStatus) String() string {
	return Stringify(s)
}

// ScheduleExportOptions represents the available ScheduleExport() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#schedule-an-export
type ScheduleExportOptions struct {
	Description *string `url:"description,omitempty" json:"description,omitempty"`
	Upload      struct {
		URL        *string `url:"url,omitempty" json:"url,omitempty"`
		HTTPMethod *string `url:"http_method,omitempty" json:"http_method,omitempty"`
	} `url:"upload,omitempty" json:"upload,omitempty"`
}

// ScheduleExport schedules a project export.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#schedule-an-export
func (s *ProjectImportExportService) ScheduleExport(pid interface{}, opt *ScheduleExportOptions, options ...RequestOptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/export", pathEscape(project))

	req, err := s.client.NewRequest(http.MethodPost, u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ExportStatus get the status of export.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#export-status
func (s *ProjectImportExportService) ExportStatus(pid interface{}, options ...RequestOptionFunc) (*ExportStatus, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/export", pathEscape(project))

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	es := new(ExportStatus)
	resp, err := s.client.Do(req, es)
	if err != nil {
		return nil, resp, err
	}

	return es, resp, err
}

// ExportDownload download the finished export.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#export-download
func (s *ProjectImportExportService) ExportDownload(pid interface{}, options ...RequestOptionFunc) ([]byte, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/export/download", pathEscape(project))

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b.Bytes(), resp, err
}

// ImportFileOptions represents the available ImportFile() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#import-a-file
type ImportFileOptions struct {
	Namespace      *string               `url:"namespace,omitempty" json:"namespace,omitempty"`
	Name           *string               `url:"name,omitempty" json:"name,omitempty"`
	File           *string               `url:"file,omitempty" json:"file,omitempty"`
	Path           *string               `url:"path,omitempty" json:"path,omitempty"`
	Overwrite      *bool                 `url:"overwrite,omitempty" json:"overwrite,omitempty"`
	OverrideParams *CreateProjectOptions `url:"override_params,omitempty" json:"override_params,omitempty"`
}

// ImportFile import a file.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#import-a-file
func (s *ProjectImportExportService) ImportFile(opt *ImportFileOptions, options ...RequestOptionFunc) (*ImportStatus, *Response, error) {
	// Open the file
	file, err := os.Open(*opt.File)
	if err != nil {
		fmt.Println(err)
	}
	// Close the file later
	defer file.Close()

	// Buffer to store our request body as bytes
	var requestBody bytes.Buffer

	// Create a multipart writer
	multiPartWriter := multipart.NewWriter(&requestBody)

	// Initialize the file field
	fileWriter, err := multiPartWriter.CreateFormFile("file", "group.*.tar.gz")
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	// Copy the actual file content to the field field's writer
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	// Populate other fields
	fw, err := multiPartWriter.CreateFormField("namespace")
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	_, err = fw.Write([]byte(*opt.Namespace))
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	fw, err = multiPartWriter.CreateFormField("path")
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	_, err = fw.Write([]byte(*opt.Path))
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	if opt.Name != nil {
		fw, err = multiPartWriter.CreateFormField("name")
		if err != nil {
			fmt.Println(err)
		}

		_, err = fw.Write([]byte(*opt.Name))
		if err != nil {
			fmt.Println(err)
		}
	}

	if opt.Overwrite != nil {
		fw, err = multiPartWriter.CreateFormField("overwrite")
		if err != nil {
			fmt.Println(err)
		}

		_, err = fw.Write([]byte(strconv.FormatBool(*opt.Overwrite)))
		if err != nil {
			fmt.Println(err)
		}
	}

	if opt.OverrideParams != nil {
		fw, err = multiPartWriter.CreateFormField("override_params")
		if err != nil {
			fmt.Println(err)
		}

		_, err = fw.Write([]byte(Stringify(opt.OverrideParams)))
		if err != nil {
			fmt.Println(err)
		}
	}
	// We completed adding the file and the fields, let's close the multipart writer
	// So it writes the ending boundary
	multiPartWriter.Close()

	req, err := s.client.NewRequest(http.MethodPost, "groups/import", nil, options)
	if err != nil {
		return nil, nil, err
	}

	// Set the buffer as the request body.
	if err = req.SetBody(&requestBody); err != nil {
		return nil, nil, err
	}

	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	// Do the request
	var status = new(ImportStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}

	return status, resp, err
}

// ImportStatus get the status of an import.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/project_import_export.html#import-status
func (s *ProjectImportExportService) ImportStatus(pid interface{}, options ...RequestOptionFunc) (*ImportStatus, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/import", pathEscape(project))

	req, err := s.client.NewRequest(http.MethodGet, u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	is := new(ImportStatus)
	resp, err := s.client.Do(req, is)
	if err != nil {
		return nil, resp, err
	}

	return is, resp, err
}
