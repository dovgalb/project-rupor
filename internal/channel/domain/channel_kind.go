package domain

type ChannelKind uint8

const (
	ChannelKindText ChannelKind = iota
	ChannelKindVoice
)

const (
	channelKindStringText  = "text"
	channelKindStringVoice = "voice"
	channelKindStringNone  = "invalid"
)

func (k ChannelKind) String() string {
	switch k {
	case ChannelKindText:
		return channelKindStringText
	case ChannelKindVoice:
		return channelKindStringVoice
	default:
		return channelKindStringNone
	}
}

func (k ChannelKind) IsValid() bool {
	return k == ChannelKindText || k == ChannelKindVoice
}

// ParseChannelKind принимает СТРОГО lowercase: "text" / "voice".
// Нормализации регистра нет — kind приходит из API-контракта, не от пользователя.
func ParseChannelKind(raw string) (ChannelKind, error) {
	switch raw {
	case channelKindStringText:
		return ChannelKindText, nil
	case channelKindStringVoice:
		return ChannelKindVoice, nil
	default:
		return 0, ErrInvalidChannelKind
	}
}
