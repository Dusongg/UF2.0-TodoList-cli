package main

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	"strconv"
)

type logNode struct {
	//add , del , update
	next      *logNode
	prev      *logNode
	operation string
	taskId    string
	oldData   interface{}
	newData   interface{}
}

var maxSize, _ = strconv.Atoi(config.Cfg.UndoLogTaskSize)

type UndoLogChain struct {
	index     int
	curSize   int
	dummyHead *logNode
	dummyTail *logNode
	client    pb.ServiceClient
}

var logChain *UndoLogChain

func NewLogChain(cli pb.ServiceClient) *UndoLogChain {
	dummyHead := new(logNode)
	dummyTail := new(logNode)
	dummyHead.next = dummyTail
	dummyTail.prev = dummyHead
	return &UndoLogChain{
		index:     0,
		curSize:   0,
		dummyHead: dummyHead,
		dummyTail: dummyTail,
		client:    cli,
	}
}

func (ulc *UndoLogChain) append(op string, oldData interface{}, newData interface{}) {
	curNode := ulc.dummyHead
	for i := 0; i < ulc.index; i++ {
		curNode = curNode.next
	}
	newNode := &logNode{
		operation: op,
		oldData:   oldData,
		newData:   newData,
		prev:      ulc.dummyHead,
		next:      curNode.next,
	}
	ulc.dummyHead.next = newNode
	curNode.next.prev = newNode
	ulc.curSize = ulc.curSize - (ulc.index) + 1
	ulc.index = 0

	for ulc.curSize > maxSize {
		ulc.dummyTail.prev.next = nil
		ulc.dummyTail = ulc.dummyTail.prev
	}

}

// 撤销
func (ulc *UndoLogChain) undo() error {
	if ulc.index == ulc.curSize {
		return nil
	}
	ulc.index++
	curNode := ulc.dummyHead
	for i := 0; i < ulc.index; i++ {
		curNode = curNode.next
	}
	switch curNode.operation {
	case "add": //撤销时为del
		if _, err := ulc.client.DelTask(context.Background(), &pb.DelTaskRequest{
			TaskNo:    curNode.newData.(*pb.AddTaskRequest).T.TaskId,
			User:      config.Cfg.Login.UserName,
			Principal: curNode.newData.(*pb.AddTaskRequest).T.Principal,
		}); err != nil {
			return err
		}
	case "del": //撤销时为add
		if _, err := ulc.client.AddTask(context.Background(), &pb.AddTaskRequest{
			T:    curNode.oldData.(*pb.Task),
			User: config.Cfg.Login.UserName,
		}); err != nil {
			return err
		}
	case "update":
		if _, err := ulc.client.ModTask(context.Background(), &pb.ModTaskRequest{
			T:    curNode.oldData.(*pb.Task),
			User: config.Cfg.Login.UserName,
		}); err != nil {
			return err
		}
	}
	return nil

}

// 重做
func (ulc *UndoLogChain) redo() error {
	if ulc.index == 0 {
		return nil
	}
	curNode := ulc.dummyHead
	for i := 0; i < ulc.index; i++ {
		curNode = curNode.next
	}
	ulc.index--
	switch curNode.operation {
	case "add": //撤销时为del
		if _, err := ulc.client.AddTask(context.Background(), curNode.newData.(*pb.AddTaskRequest)); err != nil {
			return err
		}
	case "del": //撤销时为add
		if _, err := ulc.client.DelTask(context.Background(), &pb.DelTaskRequest{
			TaskNo:    curNode.oldData.(*pb.Task).TaskId,
			User:      config.Cfg.Login.UserName,
			Principal: curNode.oldData.(*pb.Task).Principal,
		}); err != nil {
			return err
		}
	case "update":
		if _, err := ulc.client.ModTask(context.Background(), &pb.ModTaskRequest{
			T:    curNode.newData.(*pb.Task),
			User: config.Cfg.Login.UserName,
		}); err != nil {
			return err
		}
	}
	return nil

}
