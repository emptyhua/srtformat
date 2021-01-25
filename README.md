格式化srt字幕

1. 中文编码转换为UTF8
2. 时间戳格式化 0: 1: 2,342 -->  0: 1: 5,334 to 00:01:02,342 --> 00:01:05,334
3. 字幕序号强制从1开始
4. 删除UTF8 BOM

# 安装方法
```sh
go get github.com/emptyhua/srtformat
```
# 使用方法
```sh
Usage: srtformat [options] <input.srt>
options:
  -save
    	Save formated srt instead of printing out.
```
