import rest from "restler"

const restSettings = {
    headers: {
        "User-Agent": "dispatch/1.0",
    },
    timeout: 10000,
}
export const getKey = (key) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.get(`${global.config.etcdAddress}/v2/keys${key}`, copy(restSettings))
    .on("complete", (body) => {
        if (!body.node) {
            return reject(body)
        }
        resolve(body.node.value)
    })
    .on("timeout", reject)
})

export const getDirectory = (key) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.get(`${global.config.etcdAddress}/v2/keys${key}`, copy(restSettings))
    .on("complete", (body) => {
        if (!body.node) {
            return reject(body)
        }
        resolve(body.node)
    })
    .on("timeout", reject)
})


export const getRecursive = (key) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.get(`${global.config.etcdAddress}/v2/keys${key}?recursive=true&sorted=true`, copy(restSettings))
    .on("complete", (body) => {
        if (!body.node) {
            return reject(body)
        }
        resolve(body.node)
    })
    .on("timeout", reject)
})

export const watchKey = (key) => new Promise((resolve, reject) => {
    key = fixUpKey(key)
    const options = copy(restSettings)
    delete options.timeout
    rest.get(`${global.config.etcdAddress}/v2/keys${key}?wait=true`, options)
    .on("complete", (body) => {
        if (!body) {
            return watchKey(key)
        }
        if (!body.node) {
            return reject(body)
        }
        resolve(body.node)
    })
})

export const setKey = (key, value, ttl = 0) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    const options = copy(restSettings)
    options.data = { value }
    if (ttl !== 0) {
        options.data.ttl = ttl
    }
    rest.put(`${global.config.etcdAddress}/v2/keys${key}`, options)
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
    rest.post(`${global.config.etcdAddress}/v2/keys${key}`, options)
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
    rest.put(`${global.config.etcdAddress}/v2/keys${key}`, options)
    .on("complete", (body) => {resolve(body)})
    .on("timeout", reject)
})

export const deleteKey = (key, options = {}) => new Promise((resolve, reject) => {
    key = fixUpKey(key)

    rest.del(`${global.config.etcdAddress}/v2/keys${key}${options.recursive ? "?recursive=true" : ""}`, copy(restSettings))
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
