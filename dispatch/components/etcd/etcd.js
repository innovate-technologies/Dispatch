import rest from "restler"

const etcdURL = config.etcdAddress
const restSettings = {
    headers: {
        "User-Agent": "dispatch/1.0",

    },
    timeout: 10000,
}
export const getKey = (key) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.get(`${etcdURL}/v2/keys${key}`, restSettings)
    .on("complete", (body) => {
        resolve(body.node.value)
    })
    .on("timeout", reject)
})

export const setKey = (key, value, ttl = 0) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    const options = copy(restSettings)
    options.data = { value }
    if (ttl !== 0) {
        options.data.ttl = ttl
    }
    rest.put(`${etcdURL}/v2/keys${key}`, options)
    .on("complete", (body) => {resolve(body)})
    .on("timeout", reject)
})

export const postKey = (key, value, ttl = 0) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    const options = copy(restSettings)
    options.data = { value }
    if (ttl !== 0) {
        options.data.ttl = ttl
    }
    rest.post(`${etcdURL}/v2/keys${key}`, options)
    .on("complete", (body) => {resolve(body)})
    .on("timeout", reject)
})

export const refreshKey = (key, ttl) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    const options = copy(restSettings)
    options.data = {
        ttl,
        "refresh": "true",
        "prevExist": "true",
    }
    rest.put(`${etcdURL}/v2/keys${key}`, options)
    .on("complete", (body) => {resolve(body)})
    .on("timeout", reject)
})

export const deleteKey = (key, recursive = false) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.delete(`${etcdURL}/v2/keys${key}${recursive ? "?recursive=true" : ""}`, restSettings)
    .on("complete", (body) => {resolve(body)})
    .on("timeout", reject)
})

const fixUpKey = (key) => {
    if (key[0] !== "/") {
        key = "/" + key
    }
    return key
}

const copy = (object) => {
    return JSON.parse(JSON.stringify(object))
}
