/*
 *  Copyright (c) 2011 NeuStar, Inc.
 *  All rights reserved.  
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at 
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *  
 *  NeuStar, the Neustar logo and related names and logos are registered
 *  trademarks, service marks or tradenames of NeuStar, Inc. All other 
 *  product names, company names, marks, logos and symbols may be trademarks
 *  of their respective owners.
 */

package kafka

import (
  "testing"
  //"fmt"
  "bytes"
  "compress/gzip"
)

func TestMessageCreation(t *testing.T) {
  payload := []byte("testing")
  msg := NewMessage(payload)
  if msg.magic != 1 {
    t.Errorf("magic incorrect")
    t.Fail()
  }

  // generated by kafka-rb: e8 f3 5a 06
  expected := []byte{0xe8, 0xf3, 0x5a, 0x06}
  if !bytes.Equal(expected, msg.checksum[:]) {
    t.Fail()
  }
}

func TestMagic0MessageEncoding(t *testing.T) {
  // generated by kafka-rb:
  // test the old message format
  expected := []byte{0x00, 0x00, 0x00, 0x0c, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}
  length, msgsDecoded := Decode(expected, DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fail()
  }
  msgDecoded := msgsDecoded[0]

  payload := []byte("testing")
  if !bytes.Equal(payload, msgDecoded.payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatal("checksums do not match")
  }
  if msgDecoded.magic != 0 {
    t.Fatal("magic incorrect")
  }
}

func TestMessageEncoding(t *testing.T) {

  payload := []byte("testing")
  msg := NewMessage(payload)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x0d, 0x01, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}
  if !bytes.Equal(expected, msg.Encode()) {
    t.Fatalf("expected: % X\n but got: % X", expected, msg.Encode())
  }

  // verify round trip
  length, msgsDecoded := DecodeWithDefaultCodecs(msg.Encode())

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatal("checksums do not match")
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestCompressedMessageEncodingCompare(t *testing.T) {
  payload := []byte("testing")
  uncompressedMsgBytes := NewMessage(payload).Encode()
  
  msgGzipBytes := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID]).Encode()
  msgDefaultBytes := NewCompressedMessage(payload).Encode()
  if !bytes.Equal(msgDefaultBytes, msgGzipBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal", msgDefaultBytes, msgGzipBytes)
  }
}

func TestCompressedMessageEncoding(t *testing.T) {
  payload := []byte("testing")
  uncompressedMsgBytes := NewMessage(payload).Encode()
  
  msg := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID])

  expectedPayload := []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
    0xFF, 0x62, 0x60, 0x60, 0xE0, 0x65, 0x64, 0x78, 0xF1, 0x39, 0x8A,
    0xAD, 0x24, 0xB5, 0xB8, 0x24, 0x33, 0x2F, 0x1D, 0x10, 0x00, 0x00,
    0xFF, 0xFF, 0x0C, 0x6A, 0x82, 0x91, 0x11, 0x00, 0x00, 0x00}

  expectedHeader := []byte{0x00, 0x00, 0x00, 0x2F, 0x01, 0x01, 0x07, 0xFD, 0xC3, 0x76}

  expected := make([]byte, len(expectedHeader)+len(expectedPayload))
  n := copy(expected, expectedHeader)
  copy(expected[n:], expectedPayload)

  if msg.compression != 1 {
    t.Fatalf("expected compression: 1 but got: %b", msg.compression)
  }

  zipper, _ := gzip.NewReader(bytes.NewBuffer(msg.payload))
  uncompressed := make([]byte, 100)
  n, _ = zipper.Read(uncompressed)
  uncompressed = uncompressed[:n]
  zipper.Close()

  if !bytes.Equal(uncompressed, uncompressedMsgBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal", uncompressed, uncompressedMsgBytes)
  }

  if !bytes.Equal(expected, msg.Encode()) {
    t.Fatalf("expected: % X\n but got: % X", expected, msg.Encode())
  }

  // verify round trip
  length, msgsDecoded := Decode(msg.Encode(), DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatalf("checksums do not match, expected: % X but was: % X", 
      chksum, msgDecoded.checksum[:])
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestLongCompressedMessageRoundTrip(t *testing.T) {
  payloadBuf := bytes.NewBuffer([]byte{})
  // make the test bigger than buffer allocated in the Decode
  for i := 0; i < 15; i++ {
    payloadBuf.Write([]byte("testing123 "))
  }

  uncompressedMsgBytes := NewMessage(payloadBuf.Bytes()).Encode()
  msg := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID])
  
  zipper, _ := gzip.NewReader(bytes.NewBuffer(msg.payload))
  uncompressed := make([]byte, 200)
  n, _ := zipper.Read(uncompressed)
  uncompressed = uncompressed[:n]
  zipper.Close()

  if !bytes.Equal(uncompressed, uncompressedMsgBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal", 
      uncompressed, uncompressedMsgBytes)
  }

  // verify round trip
  length, msgsDecoded := Decode(msg.Encode(), DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payloadBuf.Bytes()) {
    t.Fatal("bytes not equal")
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestMultipleCompressedMessages(t *testing.T) {
  msgs := []*Message{NewMessage([]byte("testing")), 
    NewMessage([]byte("multiple")), 
    NewMessage([]byte("messages")),
  }
  msg := NewCompressedMessages(msgs...)
  
  length, msgsDecoded := DecodeWithDefaultCodecs(msg.Encode())
  if length == 0 || msgsDecoded == nil {
    t.Fatal("msgsDecoded is nil")
  }
  
  // make sure the decompressed messages match what was put in
  for index, decodedMsg := range msgsDecoded {
    if !bytes.Equal(msgs[index].payload, decodedMsg.payload) {
      t.Fatalf("Payload doesn't match, expected: % X but was: % X\n",
        msgs[index].payload, decodedMsg.payload)
    }
  }
}

func TestRequestHeaderEncoding(t *testing.T) {
  broker := newBroker("localhost:9092", "test", 0)
  request := broker.EncodeRequestHeader(REQUEST_PRODUCE)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x74, 0x65, 0x73, 0x74,
    0x00, 0x00, 0x00, 0x00}

  if !bytes.Equal(expected, request.Bytes()) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request.Bytes()))
    t.Errorf("expected: %X\n but got: %X", expected, request)
    t.Fail()
  }
}

func TestPublishRequestEncoding(t *testing.T) {
  payload := []byte("testing")
  msg := NewMessage(payload)

  pubBroker := NewBrokerPublisher("localhost:9092", "test", 0)
  request := pubBroker.broker.EncodePublishRequest(msg)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x21, 0x00, 0x00, 0x00, 0x04, 0x74, 0x65, 0x73, 0x74,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x0d,
    /* magic  comp  ......  chksum ....     ..  payload .. */
    0x01, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}

  if !bytes.Equal(expected, request) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request))
    t.Errorf("expected: % X\n but got: % X", expected, request)
    t.Fail()
  }
}

func TestConsumeRequestEncoding(t *testing.T) {

  pubBroker := NewBrokerPublisher("localhost:9092", "test", 0)
  request := pubBroker.broker.EncodeConsumeRequest(0, 1048576)

  // generated by kafka-rb, encode_request_size + encode_request
  expected := []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x01, 0x00, 0x04, 0x74,
    0x65, 0x73, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00}

  if !bytes.Equal(expected, request) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request))
    t.Errorf("expected: % X\n but got: % X", expected, request)
    t.Fail()
  }
}
