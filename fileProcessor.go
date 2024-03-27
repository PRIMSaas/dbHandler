package main

type PaymentFile struct {
	FileContent  string   `json:"fileContent"`
	ServiceCodes []string `json:"serviceCodes"`
}

type PaymentFileResponse struct {
	FileContent string `json:"fileContent"`
}

func processFileContent(content PaymentFile) (PaymentFileResponse, error) {
	// Process the file content
	// For now, just return the file content
	resp := PaymentFileResponse{
		FileContent: content.FileContent,
	}
	return resp, nil
}
