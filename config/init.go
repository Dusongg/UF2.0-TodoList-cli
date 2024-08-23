package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
)

var (
	Cfg    = NewConfig("./config/config.json")
	XLSCfg = NewXLSConfig("./config/xls_config.json")
)

type Config struct {
	Conn struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	Login struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	}
	UndoLogTaskSize string `json:"undo_log_task_size"`
	//UndoLogPatchsSize string `json:"undoLogTaskSize"`
}

type XlsConfig struct {
	Normalize_export_files struct {
		ModOrderInfo  int   `json:"修改单信息"`
		Columns1Id    []int `json:"columns1_id"`
		UpInstruction int   `json:"升级说明"`
		Columns2Id    []int `json:"columns2_id"`
	}

	Patch_export struct {
		TempPatchExport int   `json:"临时补丁导出"`
		ColumnsId       []int `json:"columns_id"`
	}
}

func NewConfig(filePath string) *Config {
	var config Config
	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Fatal(err)
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		logrus.Fatal(err)

	}
	return &config
}

func NewXLSConfig(filePath string) *XlsConfig {
	var xlsConfig XlsConfig
	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Fatal(err)
	}
	err = json.Unmarshal(data, &xlsConfig)
	if err != nil {
		logrus.Fatal(err)

	}
	//fmt.Println(xlsConfig)
	return &xlsConfig
}
