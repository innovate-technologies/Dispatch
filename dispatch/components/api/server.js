import express from "express"
const app = express()
const wrap = fn => (...args) => fn(...args).catch(args[2])


app.get("/", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))

app.get("/v1/machines", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))

app.get("/v1/machines/:name", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))

app.get("/v1/units", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))

app.get("/v1/units/:name", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))

app.put("/v1/unit", wrap((req, res) => {
    res.send("Project Dispatch API server")
}))



app.listen(config.bindIP + ":" + config.bindPort)
