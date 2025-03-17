package message

import "github.com/google/uuid"

const (
	VarControlConnected    = "control-connected"
	VarControlDisconnected = "control-disconnected"
	VarControlTerminate    = "control-terminate"
	VarControlError        = "control-error"
)

type VariantControlConnected struct {
	UserId uuid.UUID `json:"userId"`
}

type VariantControlDisconnected struct {
	UserId uuid.UUID `json:"userId"`
}

type VariantControlTerminate struct {
	Reason string `json:"reason"`
}

type VariantControlError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

func CraftControlConnected(userId uuid.UUID) *Message {
	return MustCraftMessage(VarControlConnected, VariantControlConnected{UserId: userId})
}

func CraftControlDisconnected(userId uuid.UUID) *Message {
	return MustCraftMessage(VarControlDisconnected, VariantControlDisconnected{UserId: userId})
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
