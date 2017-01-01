console.log("Project Dispatch")
console.log("Copyright 2017 Innovate Technologies")
console.log("====================================")

import os from "os"
import * as tagsUtil from "./components/utils/tags"

global.config = {
    bindIP: process.env.BINDIP || "127.0.0.1",
    bindPort: process.env.BINDPORT || "4001",
    etcdAddress: process.env.ETCDADDR || "http://127.0.0.1:2379",
    publicIP: process.env.PUBLICIP || "127.0.0.1",
    machineName: process.env.MACHINENAME || os.hostname(),
    tags: tagsUtil.parseTags(process.env.TAGS || ""),
    arch: os.arch(),
}

