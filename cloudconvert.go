package cloudconvert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

const DEFAULT_API_URL = "https://api.cloudconvert.com"

type ErrCloudConvert struct {
	Value string `json:"error"`
	Code  int    `json:"code"`
}

func (e ErrCloudConvert) String() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Value)
}

func (e ErrCloudConvert) Error() string {
	return e.String()
}

type ErrInvalidStatusCode struct {
	Expected int
	Actual   int
}

func (e ErrInvalidStatusCode) String() string {
	return fmt.Sprintf("invalid status code; expected %d but got %d", e.Expected, e.Actual)
}

func (e ErrInvalidStatusCode) Error() string {
	return e.String()
}

func invalidStatusCode(expected, actual int) ErrInvalidStatusCode {
	return ErrInvalidStatusCode{expected, actual}
}

func New(key string) (*Client, error) {
	u, err := url.Parse(DEFAULT_API_URL)
	if err != nil {
		return nil, err
	}

	c := Client{
		APIKey:    key,
		BaseURL:   DEFAULT_API_URL,
		parsedURL: *u,
	}

	return &c, nil
}

type Client struct {
	APIKey    string
	BaseURL   string
	parsedURL url.URL
}

type createProcessRequest struct {
	APIKey       string `json:"apikey"`
	InputFormat  string `json:"inputformat"`
	OutputFormat string `json:"outputformat"`
}

type createProcessResponse struct {
	URL        string `json:"url"`
	ID         string `json:"id"`
	Host       string `json:"host"`
	Expires    string `json:"expires"`
	MaxSize    int    `json:"maxsize"`
	MaxTime    int    `json:"maxtime"`
	Concurrent int    `json:"concurrent"`
	Minutes    int    `json:"minutes"`
}

func (c Client) CreateProcess(inputFormat, outputFormat string) (*Process, error) {
	reqJson, err := json.Marshal(createProcessRequest{
		c.APIKey,
		inputFormat,
		outputFormat,
	})

	if err != nil {
		return nil, err
	}

	r, err := http.DefaultClient.Post(c.BaseURL+"/process", "application/json", bytes.NewReader(reqJson))
	if err != nil {
		return nil, err
	}

	if r.StatusCode != 200 {
		var e ErrCloudConvert
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			return nil, invalidStatusCode(200, r.StatusCode)
		} else {
			return nil, e
		}
	}

	var res createProcessResponse
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return nil, err
	}

	u, err := c.parsedURL.Parse(res.URL)
	if err != nil {
		return nil, err
	}

	p := Process{
		c:   c,
		id:  res.ID,
		url: u,
	}

	return &p, nil
}

type Process struct {
	c    Client
	id   string
	url  *url.URL
	wait bool
}

func (p Process) Wait(w bool) Process {
	p.wait = w

	return p
}

func (p Process) ConvertStream(input io.Reader, filename string, outputFormat string, converterOptions map[string]string) (*ProcessStatus, error) {
	if converterOptions == nil {
		converterOptions = map[string]string{}
	}

	pr, pw := io.Pipe()

	mw := multipart.NewWriter(pw)

	done := make(chan error, 1)

	go func() {
		defer close(done)

		if err := mw.WriteField("input", "upload"); err != nil {
			done <- err
			return
		}

		if err := mw.WriteField("outputformat", outputFormat); err != nil {
			done <- err
			return
		}

		if p.wait {
			if err := mw.WriteField("wait", "true"); err != nil {
				done <- err
				return
			}
		}

		for k, v := range converterOptions {
			if err := mw.WriteField("converteroptions["+k+"]", v); err != nil {
				done <- err
				return
			}
		}

		if fw, err := mw.CreateFormFile("file", filename); err != nil {
			done <- err
			return
		} else if _, err := io.Copy(fw, input); err != nil {
			done <- err
			return
		}

		if err := mw.Close(); err != nil {
			done <- err
			return
		}

		if err := pw.Close(); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	req, err := http.NewRequest("POST", p.url.String(), pr)
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", mw.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if err := <-done; err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var e ErrCloudConvert
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, invalidStatusCode(200, res.StatusCode)
		} else {
			return nil, e
		}
	}

	var s ProcessStatus
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (p Process) Download() (io.ReadCloser, error) {
	u, err := p.url.Parse("/download/" + p.id)
	if err != nil {
		return nil, err
	}

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var e ErrCloudConvert
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, invalidStatusCode(200, res.StatusCode)
		} else {
			return nil, e
		}
	}

	return res.Body, nil
}

func (p Process) DownloadOne(f string) (io.ReadCloser, error) {
	u, err := p.url.Parse("/download/" + p.id + "/" + url.QueryEscape(f))
	if err != nil {
		return nil, err
	}

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var e ErrCloudConvert
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, invalidStatusCode(200, res.StatusCode)
		} else {
			return nil, e
		}
	}

	return res.Body, nil
}

func (p Process) Status() (*ProcessStatus, error) {
	res, err := http.Get(p.url.String())
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var e ErrCloudConvert
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, invalidStatusCode(200, res.StatusCode)
		} else {
			return nil, e
		}
	}

	var s ProcessStatus
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

type ProcessStatus struct {
	ID        string                `json:"id"`
	URL       string                `json:"url"`
	Percent   int                   `json:"percent"`
	Message   string                `json:"message"`
	Step      string                `json:"step"`
	StartTime int                   `json:"starttime"`
	EndTime   int                   `json:"endtime"`
	Expire    int                   `json:"expire"`
	Minutes   int                   `json:"minutes"`
	Group     string                `json:"group"`
	Input     *ProcessInputInfo     `json:"input"`
	Output    *ProcessOutputInfo    `json:"output"`
	Converter *ProcessConverterInfo `json:"converter"`
}

type ProcessInputInfo struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Ext      string `json:"ext"`
}

type ProcessOutputInfo struct {
	URL      string   `json:"url"`
	Size     int      `json:"size"`
	Filename string   `json:"filename"`
	Ext      string   `json:"ext"`
	Files    []string `json:"files"`
}

type ProcessConverterInfo struct {
	Format  string                 `json:"format"`
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options"`
}
