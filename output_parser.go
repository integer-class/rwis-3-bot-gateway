package main

import (
	"encoding/json"
	"github.com/pkg/errors"
)

/**
Output Types and Schemas

1. Issue Report: { "type": "issue_report", "value": "string", "meta": { "title": "string", "description": "string" } }
2. Chat: { "type": "chat", "value": "string" }
3. Personal Data Request: { "type": "personal_data_request", "value": "string" }
4. RW Data Request: { "type": "rw_data_request" }
5. Fund Data Request (Personal): { "type": "fund_data_request", "value": "string" }
6. Fund Data Request (General): { "type": "fund_data_request", "fields": ["string"] }
7. UMKM Data Request: { "type": "umkm_data_request" }
8. Broadcast Request: { "type": "broadcast_request" }
9. RT Data Request: { "type": "rt_data_request", "name": "string", "fields": ["string"] }
10. Reminder Request: { "type": "reminder_request", "before": "date", "pick": "string" }
*/

func parseGeminiAnswer(answer string) (error, map[string]interface{}) {
	// try parsing the answer to see if it's a command
	var output map[string]interface{}
	err := json.Unmarshal([]byte(answer), &output)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal gemini answer"), map[string]interface{}{}
	}
	return nil, output
}
