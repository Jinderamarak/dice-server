package message

import "github.com/google/uuid"

const (
	VarControlConnected    = "control-connected"
	VarControlDisconnected = "control-disconnected"
	VarControlTerminate    = "control-terminate"
	VarControlError        = "control-error"
)

type VariantControlConnected struct {
	UserID uuid.UUID `json:"userId"`
}

type VariantControlDisconnected struct {
	UserID uuid.UUID `json:"userId"`
}

type VariantControlTerminate struct {
	Reason string `json:"reason"`
}

type VariantControlError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

func CraftControlConnected(userID uuid.UUID) *Message {
	return MustCraftMessage(VarControlConnected, VariantControlConnected{UserID: userID})
}

func CraftControlDisconnected(userID uuid.UUID) *Message {
	return MustCraftMessage(VarControlDisconnected, VariantControlDisconnected{UserID: userID})
}

func CraftControlTerminate(reason string) *Message {
	return MustCraftMessage(VarControlTerminate, VariantControlTerminate{Reason: reason})
}

func CraftControlError(kind, message string) *Message {
	return MustCraftMessage(VarControlError, VariantControlError{Kind: kind, Message: message})
}

func IsControlMessage(message *Message) bool {
	switch message.Variant {
	case VarControlConnected, VarControlDisconnected, VarControlTerminate, VarControlError:
		return true
	default:
		return false
	}
}
