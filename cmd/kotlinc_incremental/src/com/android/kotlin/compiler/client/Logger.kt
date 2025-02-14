package com.android.kotlin.compiler.client

import org.jetbrains.kotlin.buildtools.api.KotlinLogger

class Logger : KotlinLogger {
    override val isDebugEnabled: Boolean
        get() = true

    override fun debug(msg: String) {
        println(msg)
    }

    override fun error(msg: String, throwable: Throwable?) {
        println(msg)
        if (throwable != null) {
            println(throwable)
        }
    }

    override fun info(msg: String) {
        println(msg)
    }

    override fun lifecycle(msg: String) {
        println(msg)
    }

    override fun warn(msg: String, throwable: Throwable?) {
        println(msg)
        if (throwable != null) {
            println(throwable)
        }
    }
}