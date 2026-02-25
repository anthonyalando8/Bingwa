// internal/usecase/offer/offer_service.go
package offer

import (
	"context"
	"fmt"
	"strings"

	"bingwa-service/internal/domain/offer"
)
// GenerateUSSDCode generates actual USSD code from template (uses primary USSD code, falls back to offer table)
func (s *OfferService) GenerateUSSDCode(o *offer.AgentOffer, phoneNumber string) string {
	// Determine which USSD code template to use
	var codeTemplate string
	
	// Priority 1: Use primary USSD code if available
	if o.PrimaryUSSDCode != nil && o.PrimaryUSSDCode.USSDCode != "" {
		codeTemplate = o.PrimaryUSSDCode.USSDCode
	} else {
		// Fallback: Use USSD code from offer table
		codeTemplate = o.USSDCodeTemplate
	}

	// Define placeholder replacements
	replacements := map[string]string{
		"{phone}":          phoneNumber,
		"{customer_phone}": phoneNumber,
		"{amount}":         fmt.Sprintf("%.0f", o.Amount),
		"{price}":          fmt.Sprintf("%.0f", o.Price),
	}

	// Replace all placeholders
	code := codeTemplate
	for placeholder, value := range replacements {
		code = strings.ReplaceAll(code, placeholder, value)
	}

	return code
}

// GenerateUSSDCodeWithPriority generates USSD code for a specific priority level
func (s *OfferService) GenerateUSSDCodeWithPriority(ctx context.Context, o *offer.AgentOffer, phoneNumber string, priority int) (string, error) {
	// Get USSD codes for the offer
	codes, err := s.ussdCodeRepo.GetActiveCodesByPriority(ctx, o.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get USSD codes: %w", err)
	}

	// Find the code with the specified priority
	var codeTemplate string
	for _, code := range codes {
		if code.Priority == priority && code.IsActive {
			codeTemplate = code.USSDCode
			break
		}
	}

	// If not found, use primary or fallback
	if codeTemplate == "" {
		return s.GenerateUSSDCode(o, phoneNumber), nil
	}

	// Define placeholder replacements
	replacements := map[string]string{
		"{phone}":          phoneNumber,
		"{customer_phone}": phoneNumber,
		"{amount}":         fmt.Sprintf("%.0f", o.Amount),
		"{price}":          fmt.Sprintf("%.0f", o.Price),
	}

	// Replace all placeholders
	code := codeTemplate
	for placeholder, value := range replacements {
		code = strings.ReplaceAll(code, placeholder, value)
	}

	return code, nil
}

// GenerateAllActiveUSSDCodes generates all active USSD codes for an offer (for mobile app failover)
func (s *OfferService) GenerateAllActiveUSSDCodes(ctx context.Context, o *offer.AgentOffer, phoneNumber string) ([]string, error) {
	// Get all active USSD codes sorted by priority
	codes, err := s.ussdCodeRepo.GetActiveCodesByPriority(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active USSD codes: %w", err)
	}

	// If no active codes found, use fallback from offer table
	if len(codes) == 0 {
		fallbackCode := s.GenerateUSSDCode(o, phoneNumber)
		return []string{fallbackCode}, nil
	}

	// Define placeholder replacements
	replacements := map[string]string{
		"{phone}":          phoneNumber,
		"{customer_phone}": phoneNumber,
		"{amount}":         fmt.Sprintf("%.0f", o.Amount),
		"{price}":          fmt.Sprintf("%.0f", o.Price),
	}

	// Generate all active codes
	generatedCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		if !code.IsActive {
			continue
		}

		ussdCode := code.USSDCode
		for placeholder, value := range replacements {
			ussdCode = strings.ReplaceAll(ussdCode, placeholder, value)
		}
		generatedCodes = append(generatedCodes, ussdCode)
	}

	return generatedCodes, nil
}

// GetUSSDCodeForExecution returns the USSD code to execute (with metadata for mobile app)
func (s *OfferService) GetUSSDCodeForExecution(ctx context.Context, agentID, offerID int64, phoneNumber string) (*offer.USSDCodeExecutionInfo, error) {
	// Get offer
	o, err := s.GetOffer(ctx, agentID, offerID)
	if err != nil {
		return nil, err
	}

	// Get all active USSD codes
	codes, err := s.ussdCodeRepo.GetActiveCodesByPriority(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get USSD codes: %w", err)
	}

	// If no codes, use fallback
	if len(codes) == 0 {
		fallbackCode := s.GenerateUSSDCode(o, phoneNumber)
		return &offer.USSDCodeExecutionInfo{
			USSDCode:         fallbackCode,
			ProcessingType:   o.USSDProcessingType,
			ExpectedResponse: o.USSDExpectedResponse.String,
			ErrorPattern:     o.USSDErrorPattern.String,
			IsFallback:       true,
		}, nil
	}

	// Use primary code (priority 1)
	primaryCode := codes[0]
	
	// Generate USSD code with placeholders replaced
	replacements := map[string]string{
		"{phone}":          phoneNumber,
		"{customer_phone}": phoneNumber,
		"{amount}":         fmt.Sprintf("%.0f", o.Amount),
		"{price}":          fmt.Sprintf("%.0f", o.Price),
	}

	ussdCode := primaryCode.USSDCode
	for placeholder, value := range replacements {
		ussdCode = strings.ReplaceAll(ussdCode, placeholder, value)
	}

	// Prepare fallback codes
	fallbackCodes := make([]string, 0, len(codes)-1)
	for i := 1; i < len(codes); i++ {
		fallbackUSSD := codes[i].USSDCode
		for placeholder, value := range replacements {
			fallbackUSSD = strings.ReplaceAll(fallbackUSSD, placeholder, value)
		}
		fallbackCodes = append(fallbackCodes, fallbackUSSD)
	}

	return &offer.USSDCodeExecutionInfo{
		USSDCodeID:       primaryCode.ID,
		USSDCode:         ussdCode,
		ProcessingType:   primaryCode.ProcessingType,
		ExpectedResponse: primaryCode.ExpectedResponse.String,
		ErrorPattern:     primaryCode.ErrorPattern.String,
		SignaturePattern: primaryCode.SignaturePattern.String,
		Priority:         primaryCode.Priority,
		FallbackCodes:    fallbackCodes,
		IsFallback:       false,
	}, nil
}