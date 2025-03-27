package uuidutil

import "github.com/google/uuid"

func UnmarshallJSONUUID(data []byte) (uuid.UUID, error) {
	parsedUUID, err := uuid.Parse(string(data))
	if err != nil {
		return uuid.Nil, err
	}

	return parsedUUID, nil
}
