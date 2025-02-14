package com.android.kotlin.compiler.client

import java.io.Serializable
import java.rmi.RemoteException
import java.rmi.server.UnicastRemoteObject
import org.jetbrains.kotlin.build.report.metrics.BuildMetrics
import org.jetbrains.kotlin.build.report.metrics.BuildMetricsReporterImpl
import org.jetbrains.kotlin.build.report.metrics.GradleBuildPerformanceMetric
import org.jetbrains.kotlin.build.report.metrics.GradleBuildTime
import org.jetbrains.kotlin.buildtools.api.KotlinLogger
import org.jetbrains.kotlin.daemon.common.CompilationResultCategory
import org.jetbrains.kotlin.daemon.common.CompilationResults
import org.jetbrains.kotlin.daemon.common.CompileIterationResult
import org.jetbrains.kotlin.daemon.common.SOCKET_ANY_FREE_PORT

class CompilationResults(
    private val log: KotlinLogger
) : CompilationResults,
    UnicastRemoteObject(SOCKET_ANY_FREE_PORT, null, null) {

    var icLogLines: List<String> = emptyList()
    private val buildMetricsReporter =
        BuildMetricsReporterImpl<GradleBuildTime, GradleBuildPerformanceMetric>()
    val buildMetrics: BuildMetrics<GradleBuildTime, GradleBuildPerformanceMetric>
        get() = buildMetricsReporter.getMetrics()

    @Throws(RemoteException::class)
    override fun add(compilationResultCategory: Int, value: Serializable) {
        when (compilationResultCategory) {
            CompilationResultCategory.IC_COMPILE_ITERATION.code -> {
                val compileIterationResult = value as? CompileIterationResult
                if (compileIterationResult != null) {
                    val sourceFiles = compileIterationResult.sourceFiles
                    if (sourceFiles.any()) {
                        log.debug("compile iteration: ${sourceFiles.joinToString()}")
                        buildMetrics.buildPerformanceMetrics.add(GradleBuildPerformanceMetric.COMPILE_ITERATION)
                    }
                    val exitCode = compileIterationResult.exitCode
                    log.debug("compiler exit code: $exitCode")
                }
            }

            CompilationResultCategory.BUILD_REPORT_LINES.code,
            CompilationResultCategory.VERBOSE_BUILD_REPORT_LINES.code -> {
                @Suppress("UNCHECKED_CAST")
                (value as? List<String>)?.let { icLogLines = it }
            }

            CompilationResultCategory.BUILD_METRICS.code -> {
                @Suppress("UNCHECKED_CAST")
                (value as? BuildMetrics<GradleBuildTime, GradleBuildPerformanceMetric>)?.let {
                    buildMetricsReporter.addMetrics(
                        it
                    )
                }
            }
        }
    }

    fun release() {
        unexportObject(this, true)
    }
}