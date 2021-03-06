// Copyright 2014, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binlog

import (
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/youtube/vitess/go/sync2"
	"github.com/youtube/vitess/go/vt/binlog/proto"
	"github.com/youtube/vitess/go/vt/mysqlctl"
	myproto "github.com/youtube/vitess/go/vt/mysqlctl/proto"
)

// sample Google MySQL event data
var (
	rotateEvent   = []byte{0x0, 0x0, 0x0, 0x0, 0x4, 0x88, 0xf3, 0x0, 0x0, 0x33, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0, 0x23, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x76, 0x74, 0x2d, 0x30, 0x30, 0x30, 0x30, 0x30, 0x36, 0x32, 0x33, 0x34, 0x34, 0x2d, 0x62, 0x69, 0x6e, 0x2e, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31}
	formatEvent   = []byte{0x98, 0x68, 0xe9, 0x53, 0xf, 0x88, 0xf3, 0x0, 0x0, 0x66, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x35, 0x2e, 0x31, 0x2e, 0x36, 0x33, 0x2d, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2d, 0x6c, 0x6f, 0x67, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1b, 0x38, 0xd, 0x0, 0x8, 0x0, 0x12, 0x0, 0x4, 0x4, 0x4, 0x4, 0x12, 0x0, 0x0, 0x53, 0x0, 0x4, 0x1a, 0x8, 0x0, 0x0, 0x0, 0x8, 0x8, 0x8, 0x2}
	beginEvent    = []byte{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x58, 0x0, 0x0, 0x0, 0xc2, 0x0, 0x0, 0x0, 0x8, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x42, 0x45, 0x47, 0x49, 0x4e}
	commitEvent   = []byte{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x59, 0x0, 0x0, 0x0, 0xc2, 0x0, 0x0, 0x0, 0x8, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54}
	rollbackEvent = []byte{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x5b, 0x0, 0x0, 0x0, 0xc2, 0x0, 0x0, 0x0, 0x8, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x52, 0x4f, 0x4c, 0x4c, 0x42, 0x41, 0x43, 0x4b}
	insertEvent   = []byte{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x9f, 0x0, 0x0, 0x0, 0x61, 0x1, 0x0, 0x0, 0x0, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x20, 0x69, 0x6e, 0x74, 0x6f, 0x20, 0x76, 0x74, 0x5f, 0x61, 0x28, 0x65, 0x69, 0x64, 0x2c, 0x20, 0x69, 0x64, 0x29, 0x20, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x20, 0x28, 0x31, 0x2c, 0x20, 0x31, 0x29, 0x20, 0x2f, 0x2a, 0x20, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x20, 0x76, 0x74, 0x5f, 0x61, 0x20, 0x28, 0x65, 0x69, 0x64, 0x20, 0x69, 0x64, 0x20, 0x29, 0x20, 0x28, 0x31, 0x20, 0x31, 0x20, 0x29, 0x3b, 0x20, 0x2a, 0x2f}
	createEvent   = []byte{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0xca, 0x0, 0x0, 0x0, 0xed, 0x3, 0x0, 0x0, 0x0, 0x0, 0xa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x8, 0x0, 0x8, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x20, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x20, 0x69, 0x66, 0x20, 0x6e, 0x6f, 0x74, 0x20, 0x65, 0x78, 0x69, 0x73, 0x74, 0x73, 0x20, 0x76, 0x74, 0x5f, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x20, 0x28, 0xa, 0x69, 0x64, 0x20, 0x62, 0x69, 0x67, 0x69, 0x6e, 0x74, 0x20, 0x61, 0x75, 0x74, 0x6f, 0x5f, 0x69, 0x6e, 0x63, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2c, 0xa, 0x6d, 0x73, 0x67, 0x20, 0x76, 0x61, 0x72, 0x63, 0x68, 0x61, 0x72, 0x28, 0x36, 0x34, 0x29, 0x2c, 0xa, 0x70, 0x72, 0x69, 0x6d, 0x61, 0x72, 0x79, 0x20, 0x6b, 0x65, 0x79, 0x20, 0x28, 0x69, 0x64, 0x29, 0xa, 0x29, 0x20, 0x45, 0x6e, 0x67, 0x69, 0x6e, 0x65, 0x3d, 0x49, 0x6e, 0x6e, 0x6f, 0x44, 0x42}
	xidEvent      = []byte{0x98, 0x68, 0xe9, 0x53, 0x10, 0x88, 0xf3, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x4e, 0xa, 0x0, 0x0, 0x0, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x78, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
)

func sendTestEvents(channel chan<- proto.BinlogEvent, events [][]byte) {
	for _, buf := range events {
		channel <- mysqlctl.NewGoogleBinlogEvent(buf)
	}
	close(channel)
}

func TestBinlogConnStreamerParseEventsXID(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	want := []proto.BinlogTransaction{
		proto.BinlogTransaction{
			Statements: []proto.Statement{
				proto.Statement{Category: proto.BL_SET, Sql: []byte("SET TIMESTAMP=1407805592")},
				proto.Statement{Category: proto.BL_DML, Sql: []byte("insert into vt_a(eid, id) values (1, 1) /* _stream vt_a (eid id ) (1 1 ); */")},
			},
			Timestamp: 1407805592,
			GTIDField: myproto.GTIDField{
				Value: myproto.GoogleGTID{GroupID: 0x0d}},
		},
	}
	var got []proto.BinlogTransaction
	sendTransaction := func(trans *proto.BinlogTransaction) error {
		got = append(got, *trans)
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	bls.svm.Wait()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("binlogConnStreamer.parseEvents(): got %#v, want %#v", got, want)
	}
}

func TestBinlogConnStreamerParseEventsCommit(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		commitEvent,
	}

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	want := []proto.BinlogTransaction{
		proto.BinlogTransaction{
			Statements: []proto.Statement{
				proto.Statement{Category: proto.BL_SET, Sql: []byte("SET TIMESTAMP=1407805592")},
				proto.Statement{Category: proto.BL_DML, Sql: []byte("insert into vt_a(eid, id) values (1, 1) /* _stream vt_a (eid id ) (1 1 ); */")},
			},
			Timestamp: 1407805592,
			GTIDField: myproto.GTIDField{
				Value: myproto.GoogleGTID{GroupID: 0x0d}},
		},
	}
	var got []proto.BinlogTransaction
	sendTransaction := func(trans *proto.BinlogTransaction) error {
		got = append(got, *trans)
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	bls.svm.Wait()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("binlogConnStreamer.parseEvents(): got %#v, want %#v", got, want)
	}
}

func TestBinlogConnStreamerStop(t *testing.T) {
	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return nil
	}

	// Start parseEvents(), but don't send it anything, so it just waits.
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		bls.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Errorf("timed out waiting for binlogConnStreamer.Stop()")
	}
}

func TestBinlogConnStreamerParseEventsSendEOF(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}
	want := io.EOF

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return io.EOF
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if err != want {
			t.Errorf("wrong error, got %#v, want %#v", err, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsSendErrorXID(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}
	want := "send reply error: foobar"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return fmt.Errorf("foobar")
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsSendErrorCommit(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		commitEvent,
	}
	want := "send reply error: foobar"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return fmt.Errorf("foobar")
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsInvalid(t *testing.T) {
	invalidEvent := rotateEvent[:19]

	input := [][]byte{
		invalidEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}
	want := "can't parse binlog event, invalid data: mysqlctl.googleBinlogEvent{binlogEvent:mysqlctl.binlogEvent{0x0, 0x0, 0x0, 0x0, 0x4, 0x88, 0xf3, 0x0, 0x0, 0x33, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0}}"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsInvalidFormat(t *testing.T) {
	invalidEvent := make([]byte, len(formatEvent))
	copy(invalidEvent, formatEvent)
	invalidEvent[19+2+50+4] = 12 // mess up the HeaderLength

	input := [][]byte{
		rotateEvent,
		invalidEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}
	want := "can't parse FORMAT_DESCRIPTION_EVENT: header length = 12, should be >= 19, event data: mysqlctl.googleBinlogEvent{binlogEvent:mysqlctl.binlogEvent{0x98, 0x68, 0xe9, 0x53, 0xf, 0x88, 0xf3, 0x0, 0x0, 0x66, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x35, 0x2e, 0x31, 0x2e, 0x36, 0x33, 0x2d, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2d, 0x6c, 0x6f, 0x67, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc, 0x38, 0xd, 0x0, 0x8, 0x0, 0x12, 0x0, 0x4, 0x4, 0x4, 0x4, 0x12, 0x0, 0x0, 0x53, 0x0, 0x4, 0x1a, 0x8, 0x0, 0x0, 0x0, 0x8, 0x8, 0x8, 0x2}}"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsNoFormat(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		//formatEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}
	want := "got a real event before FORMAT_DESCRIPTION_EVENT: mysqlctl.googleBinlogEvent{binlogEvent:mysqlctl.binlogEvent{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x58, 0x0, 0x0, 0x0, 0xc2, 0x0, 0x0, 0x0, 0x8, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x42, 0x45, 0x47, 0x49, 0x4e}}"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsInvalidQuery(t *testing.T) {
	invalidEvent := make([]byte, len(insertEvent))
	copy(invalidEvent, insertEvent)
	invalidEvent[19+8+4+4] = 200 // mess up the db_name length

	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		invalidEvent,
		xidEvent,
	}
	want := "can't get query from binlog event: SQL query position = 240, which is outside buffer, event data: mysqlctl.googleBinlogEvent{binlogEvent:mysqlctl.binlogEvent{0x98, 0x68, 0xe9, 0x53, 0x2, 0x88, 0xf3, 0x0, 0x0, 0x9f, 0x0, 0x0, 0x0, 0x61, 0x1, 0x0, 0x0, 0x0, 0x0, 0xd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc8, 0x0, 0x0, 0x1a, 0x0, 0x0, 0x0, 0x40, 0x0, 0x0, 0x1, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x21, 0x0, 0x76, 0x74, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x70, 0x61, 0x63, 0x65, 0x0, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x20, 0x69, 0x6e, 0x74, 0x6f, 0x20, 0x76, 0x74, 0x5f, 0x61, 0x28, 0x65, 0x69, 0x64, 0x2c, 0x20, 0x69, 0x64, 0x29, 0x20, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x20, 0x28, 0x31, 0x2c, 0x20, 0x31, 0x29, 0x20, 0x2f, 0x2a, 0x20, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x20, 0x76, 0x74, 0x5f, 0x61, 0x20, 0x28, 0x65, 0x69, 0x64, 0x20, 0x69, 0x64, 0x20, 0x29, 0x20, 0x28, 0x31, 0x20, 0x31, 0x20, 0x29, 0x3b, 0x20, 0x2a, 0x2f}}"

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	sendTransaction := func(trans *proto.BinlogTransaction) error {
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err == nil {
			t.Errorf("expected error, got none")
			return
		}
		if got := err.Error(); got != want {
			t.Errorf("wrong error, got %#v, want %#v", got, want)
		}
	})
	bls.svm.Wait()
}

func TestBinlogConnStreamerParseEventsRollback(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		beginEvent,
		insertEvent,
		insertEvent,
		rollbackEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	want := []proto.BinlogTransaction{
		proto.BinlogTransaction{
			Statements: []proto.Statement{
				proto.Statement{Category: proto.BL_SET, Sql: []byte("SET TIMESTAMP=1407805592")},
				proto.Statement{Category: proto.BL_DML, Sql: []byte("insert into vt_a(eid, id) values (1, 1) /* _stream vt_a (eid id ) (1 1 ); */")},
			},
			Timestamp: 1407805592,
			GTIDField: myproto.GTIDField{
				Value: myproto.GoogleGTID{GroupID: 0x0d}},
		},
	}
	var got []proto.BinlogTransaction
	sendTransaction := func(trans *proto.BinlogTransaction) error {
		got = append(got, *trans)
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	bls.svm.Wait()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("binlogConnStreamer.parseEvents(): got %#v, want %#v", got, want)
	}
}

func TestBinlogConnStreamerParseEventsCreate(t *testing.T) {
	input := [][]byte{
		rotateEvent,
		formatEvent,
		createEvent,
		beginEvent,
		insertEvent,
		xidEvent,
	}

	bls := newBinlogConnStreamer("", nil).(*binlogConnStreamer)
	events := make(chan proto.BinlogEvent)

	want := []proto.BinlogTransaction{
		proto.BinlogTransaction{
			Statements: []proto.Statement{
				proto.Statement{Category: proto.BL_SET, Sql: []byte("SET TIMESTAMP=1407805592")},
				proto.Statement{Category: proto.BL_DDL, Sql: []byte("create table if not exists vt_insert_test (\nid bigint auto_increment,\nmsg varchar(64),\nprimary key (id)\n) Engine=InnoDB")},
			},
			Timestamp: 1407805592,
			GTIDField: myproto.GTIDField{
				Value: myproto.GoogleGTID{GroupID: 0x0a}},
		},
		proto.BinlogTransaction{
			Statements: []proto.Statement{
				proto.Statement{Category: proto.BL_SET, Sql: []byte("SET TIMESTAMP=1407805592")},
				proto.Statement{Category: proto.BL_DML, Sql: []byte("insert into vt_a(eid, id) values (1, 1) /* _stream vt_a (eid id ) (1 1 ); */")},
			},
			Timestamp: 1407805592,
			GTIDField: myproto.GTIDField{
				Value: myproto.GoogleGTID{GroupID: 0x0d}},
		},
	}
	var got []proto.BinlogTransaction
	sendTransaction := func(trans *proto.BinlogTransaction) error {
		got = append(got, *trans)
		return nil
	}

	go sendTestEvents(events, input)
	bls.svm.Go(func(svm *sync2.ServiceManager) {
		err := bls.parseEvents(events, sendTransaction)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	bls.svm.Wait()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("binlogConnStreamer.parseEvents(): got %#v, want %#v", got, want)
	}
}
