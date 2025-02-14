package com.android.kotlin.compiler.client

import org.jetbrains.kotlin.cli.common.messages.CompilerMessageSeverity
import org.jetbrains.kotlin.cli.common.messages.CompilerMessageSourceLocation
import org.jetbrains.kotlin.cli.common.messages.MessageCollector

class MessageCollectorImpl : MessageCollector {
    var hasErrors = false
    override fun clear() {}

    override fun report(
        severity: CompilerMessageSeverity,
        message: String,
        location: CompilerMessageSourceLocation?,
    ) {
        if (severity.isError) {
            hasErrors = true
        }
        println("${severity.name}\t${location?.path ?: ""}:${location?.line ?: ""} \t$message")
    }

    override fun hasErrors() = hasErrors
}