package shortdrama

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"time"
)

const xBogusAlphabet = "Dkdpgh4ZKsQB80/Mfvw36XI1R25-WUAlEi7NLboqYTOPuzmFjJnryx9HVGcaStCe"

func signTikTokURL(target *url.URL, userAgent string, timestamp time.Time) string {
	query := target.RawQuery
	signature := generateXBogus(query, userAgent, timestamp.Unix())
	if query == "" {
		return target.String() + "?X-Bogus=" + url.QueryEscape(signature)
	}
	return target.String() + "&X-Bogus=" + url.QueryEscape(signature)
}

func generateXBogus(query, userAgent string, timestamp int64) string {
	paramsHash := doubleMD5([]byte(query))
	bodyHash := doubleMD5(nil)
	userAgentCipher := rc4Bytes([]byte{0, 1, 14}, []byte(userAgent))
	encodedUserAgent := base64.StdEncoding.EncodeToString(userAgentCipher)
	userAgentHash := md5.Sum([]byte(encodedUserAgent))

	payload := make([]byte, 0, 19)
	payload = append(payload, 0x40, 0x00, 0x01, 0x0e)
	payload = append(payload, paramsHash[14], paramsHash[15])
	payload = append(payload, bodyHash[14], bodyHash[15])
	payload = append(payload, userAgentHash[14], userAgentHash[15])
	timeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeBytes, uint32(timestamp))
	payload = append(payload, timeBytes...)
	magicBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(magicBytes, 0x4a41279f)
	payload = append(payload, magicBytes...)
	checksum := byte(0)
	for _, value := range payload {
		checksum ^= value
	}
	payload = append(payload, checksum)

	encoded := append([]byte{0x02, 0xff}, rc4Bytes([]byte{0xff}, payload)...)
	return base64.NewEncoding(xBogusAlphabet).EncodeToString(encoded)
}

func doubleMD5(data []byte) [md5.Size]byte {
	first := md5.Sum(data)
	return md5.Sum(first[:])
}

func rc4Bytes(key, data []byte) []byte {
	state := make([]byte, 256)
	for index := range state {
		state[index] = byte(index)
	}
	position := 0
	for index := range state {
		position = (position + int(state[index]) + int(key[index%len(key)])) % len(state)
		state[index], state[position] = state[position], state[index]
	}

	result := make([]byte, len(data))
	left, right := 0, 0
	for index, value := range data {
		left = (left + 1) % len(state)
		right = (right + int(state[left])) % len(state)
		state[left], state[right] = state[right], state[left]
		result[index] = value ^ state[(int(state[left])+int(state[right]))%len(state)]
	}
	return result
}
