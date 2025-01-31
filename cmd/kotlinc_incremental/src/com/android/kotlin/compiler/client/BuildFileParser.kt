package com.android.kotlin.compiler.client

import org.xml.sax.Attributes
import org.xml.sax.helpers.DefaultHandler

class BuildFileParser : DefaultHandler() {
    val classpaths: List<String>
        get() = _classpaths
    val sources: List<String>
        get() = _sources
    val javaSources: List<String>
        get() = _javaSources
    val moduleName: String?
        get() = _moduleName
    val outputDirName: String?
        get() = _outputDirName

    private val _classpaths = mutableListOf<String>()
    private val _sources = mutableListOf<String>()
    private val _javaSources = mutableListOf<String>()
    private var _moduleName: String? = null
    private var _outputDirName: String? = null

    override fun startElement(
        uri: String?,
        localName: String?,
        qName: String?,
        attributes: Attributes?
    ) {
        when(qName) {
            "module" -> parseModule(attributes)
            "classpath" -> parseClassPath(attributes)
            "sources" -> parseSources(attributes)
            "javaSourceRoots" -> parseJavaSourceRoots(attributes)
        }
    }

    private fun parseClassPath(attributes: Attributes?) {
        if (attributes == null) {
            return
        }

        val cp = attributes.getValue("", "path")
        if (cp == null) {
            return
        }
        _classpaths.add(cp)
    }

    private fun parseSources(attributes: Attributes?) {
        if (attributes == null) {
            return
        }

        val path = attributes.getValue("", "path")
        if (path == null) {
            return
        }

        _sources.add(path)
    }

    private fun parseJavaSourceRoots(attributes: Attributes?) {
        if (attributes == null) {
            return
        }

        val path = attributes.getValue("", "path")
        if (path == null) {
            return
        }

        _javaSources.add(path)
    }

    private fun parseModule(attributes: Attributes?) {
        if (attributes == null) {
            return
        }

        _moduleName = attributes.getValue("", "name")
        _outputDirName = attributes.getValue("", "outputDir")
    }
}