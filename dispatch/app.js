/**
 * This file sets essential global variables and then bootstraps the app
 * after enabling ES6-syntax using Babel.
 *
 * Only ES5-compatible syntax should be used in this file,
 * as Babel hasn't been loaded yet. Keep this file slim as its sole role
 * is to set up essential globals and bootstrap the app.
 */


require("babel-polyfill");
require("babel-register");
require("./dispatch");
