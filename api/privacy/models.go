package privacy

import (
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/provide-go/api"
)

// Circuit model
type Circuit struct {
	*api.Model

	Name          *string `json:"name"`
	Description   *string `json:"description"`
	Identifier    *string `json:"identifier"`
	Provider      *string `json:"provider"`
	ProvingScheme *string `json:"proving_scheme"`
	Curve         *string `json:"curve"`
	Status        *string `json:"status"`

	NoteStoreID      *uuid.UUID `json:"note_store_id"`
	NullifierStoreID *uuid.UUID `json:"nullifier_store_id"`

	Artifacts        map[string]interface{} `json:"artifacts,omitempty"`
	VerifierContract map[string]interface{} `json:"verifier_contract,omitempty"`
}

// StoreValueResponse model
type StoreValueResponse struct {
	Errors       []*api.Error           `json:"errors,omitempty"`
	Length       *int                   `json:"length,omitempty"`
	Root         *string                `json:"root,omitempty"`
	NullifierKey *string                `json:"nullifier_key,omitempty"`
	Value        *string                `json:"value"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProveResponse model
type ProveResponse struct {
	Errors []*api.Error `json:"errors,omitempty"`
	Proof  *string      `json:"proof"`
}

// VerificationResponse model
type VerificationResponse struct {
	Errors []*api.Error `json:"errors,omitempty"`
	Result bool         `json:"result"`
}
