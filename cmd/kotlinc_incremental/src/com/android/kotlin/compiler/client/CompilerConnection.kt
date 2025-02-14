package com.android.kotlin.compiler.client


import java.io.File
import org.jetbrains.kotlin.cli.common.messages.CompilerMessageSeverity
import org.jetbrains.kotlin.cli.common.messages.MessageCollector
import org.jetbrains.kotlin.daemon.client.CompileServiceSession
import org.jetbrains.kotlin.daemon.client.DaemonReportMessage
import org.jetbrains.kotlin.daemon.client.DaemonReportingTargets
import org.jetbrains.kotlin.daemon.client.KotlinCompilerClient
import org.jetbrains.kotlin.daemon.common.CompilerId
import org.jetbrains.kotlin.daemon.common.DaemonJVMOptions
import org.jetbrains.kotlin.daemon.common.DaemonOptions
import org.jetbrains.kotlin.daemon.common.DaemonReportCategory
import org.jetbrains.kotlin.daemon.common.makeAutodeletingFlagFile
import org.jetbrains.kotlin.daemon.common.runFilesPathOrDefault


class CompilerConnection(
    classPath: List<String>,
    private val messageCollector: MessageCollector,
    private val daemonOptions: DaemonOptions,
    private val jvmOptions: DaemonJVMOptions,
) {
    private val compilerId = CompilerId(compilerClasspath = classPath)

    val session: CompileServiceSession? by lazy {
        val baseDir = File(daemonOptions.runFilesPathOrDefault)
        val clientAliveFlagFile = makeAutodeletingFlagFile(
            keyword = "compiler-client", baseDir = baseDir
        )

        println("client: " + clientAliveFlagFile.absolutePath)
        // TODO: investigate that multiple clients don't result in multiple daemons
        val sessionAliveFlagFile =
            makeAutodeletingFlagFile(
                keyword = "compiler-session", baseDir = baseDir
            )

        println("session: " + sessionAliveFlagFile.absolutePath)

        val daemonReportMessages = ArrayList<DaemonReportMessage>()
        val daemonReportingTargets = DaemonReportingTargets(messages = daemonReportMessages)
        val connection = KotlinCompilerClient.connectAndLease(
            compilerId,
            clientAliveFlagFile,
            jvmOptions,
            daemonOptions,
            daemonReportingTargets,
            autostart = true,
            leaseSession = true,
            sessionAliveFlagFile = sessionAliveFlagFile
        )

        if (connection == null) {
            println("connection is null")
        } else {
            println("connection is not null but still not working")
        }
        println("printing daemon reports")
        for (message in daemonReportMessages) {
            val severity = when (message.category) {
                DaemonReportCategory.DEBUG -> CompilerMessageSeverity.LOGGING
                DaemonReportCategory.INFO -> CompilerMessageSeverity.INFO
                DaemonReportCategory.EXCEPTION -> CompilerMessageSeverity.EXCEPTION
            }
            messageCollector.report(severity, message.message)
        }

        connection

        // TODO: investigate just using KotlinCompilerClient#connectAndLease directly.
//        KotlinCompilerRunnerUtils.newDaemonConnection(
//            compilerId,
//            clientAliveFlagFile,
//            sessionAliveFlagFile,
//            messageCollector,
//            true,
//            daemonOptions,
//            jvmOptions,
//        )
    }

}