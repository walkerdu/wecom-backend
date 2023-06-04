# 企业微信ChatGPT聊天应用
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?logoWidth=40)](https://opensource.org/licenses/MIT)

本项目是一个基于Go开发的企业微信聊天应用，目前只支持OpenAI的Chat服务，用户通过企业微信的应用发送聊天后，后端服务会通过SSE异步推流的方式和OpenAI交互，直到所有数据接收完后，将OpenAI的生成结果推送给企业微信的用户，如下：

<img width="391" alt="image" src="https://github.com/walkerdu/wecom-backend/assets/5126855/0bdc4a49-d61d-427d-858b-4f5043465f46">

## Usage
1. **编译**
```bash
$git clone https://github.com/walkerdu/wecom-backend.git
$cd wecom-backend
$make
```
2. **基于`configs/config.json`修改实际参数**
```json
{
    "open_ai": {
        "api_key": "sk-xxxxxxx"
    },
    "we_com": {
        "agent_config": {
            "corp_id": "ww123456",
            "agent_id": 1000004,
            "agent_secret": "Vitug6o-xxxx",
            "agent_token": "8kxL1xxxxxx",
            "agent_encoding_aes_key": "nxyGtXNFKzj7OHytzWkEV9awgxxxxxx"
        },
        "addr": ":9001"
    }
}
```
3. **启动服务**
```bash
$bin/wecom-backend -f configs/config.json
```
也可以直接通过命令行传入服务参数，如下：
```bash
$bin/wecom-backend --corp_id ww2712xxx --agent_id 1000004 --agent_secret Vitug6o-xxxx --agent_token 8kxLxxxxx --agent_encoding_aes_key nxyGtXNFKzj7xxxxxxxxx --addr :9001 --openai_apikey sk-80apwArF4xxxxxxx
```
