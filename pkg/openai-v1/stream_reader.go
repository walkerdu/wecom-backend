// 参考：https://github.com/walkerdu/go-openai/blob/master/stream_reader.go
package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
)

var (
	ErrTooManyEmptyStreamMessages = errors.New("stream has sent too many empty messages")
)

type streamReader struct {
	isFinished bool

	reader   *bufio.Reader
	response *ChatCompletionRsp
}

func (stream *streamReader) Recv() (err error) {
	if stream.isFinished {
		err = io.EOF
		return
	}

	var emptyMessagesCount uint

waitForData:
	line, err := stream.reader.ReadBytes('\n')
	if err != nil {
		return
	}

	var headerData = []byte("data: ")
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, headerData) {
		err = fmt.Errorf("error, data no prefix \"data:")
		emptyMessagesCount++
		if emptyMessagesCount > 10 {
			err = ErrTooManyEmptyStreamMessages
			return
		}

		goto waitForData
	}

	line = bytes.TrimPrefix(line, headerData)
	if string(line) == "[DONE]" {
		stream.isFinished = true
		err = io.EOF
		return
	}

	if err = json.Unmarshal(line, stream.response); nil != err {
		log.Printf("[ERROR][handleChatMessage]Unmarshal failed err=%s", err)
		return
	}

	return
}

//func (stream *streamReader) Close() {
//	stream.response.Body.Close()
//}
