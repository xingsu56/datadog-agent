// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog
// (https://www.datadoghq.com/).
// Copyright 2019 Datadog, Inc.

#include "cgo_free.h"
#include "sixstrings.h"
#include <tagger.h>

// these must be set by the Agent
static cb_get_tags_t cb_get_tags = NULL;

PyObject *get_tags(PyObject *self, PyObject *args)
{
    if (cb_get_tags == NULL)
        Py_RETURN_NONE;

    PyGILState_STATE gstate = PyGILState_Ensure();

    char *id;
    int highCard;
    if (!PyArg_ParseTuple(args, "si", &id, &highCard)) {
        PyErr_SetString(PyExc_TypeError, "wrong parameters type");
        PyGILState_Release(gstate);
        // we need to return NULL to raise the exception set by PyErr_SetString
        return NULL;
    }

    char **tags = cb_get_tags(id, highCard);

    PyGILState_Release(gstate);
    if (tags == NULL) {
        Py_RETURN_NONE;
    }

    PyObject *res = PyList_New(0);
    int i;
    for (i = 0; tags[i]; i++) {
        PyObject *pyTag = PyStringFromCString(tags[i]);
        cgo_free(tags[i]);
        PyList_Append(res, pyTag);
    }
    cgo_free(tags);
    return res;
}

void _set_get_tags_cb(cb_get_tags_t cb)
{
    cb_get_tags = cb;
}

static PyMethodDef methods[] = {
    { "get_tags", (PyCFunction)get_tags, METH_VARARGS, "Get tags for an entity." }, { NULL, NULL } // guards
};

#ifdef DATADOG_AGENT_THREE
static struct PyModuleDef module_def = { PyModuleDef_HEAD_INIT, TAGGER_MODULE_NAME, NULL, -1, methods };

PyMODINIT_FUNC PyInit_tagger(void)
{
    return PyModule_Create(&module_def);
}
#endif

#ifdef DATADOG_AGENT_TWO
// in Python2 keep the object alive for the program lifetime
static PyObject *module;

void Py2_init_tagger()
{
    module = Py_InitModule(TAGGER_MODULE_NAME, methods);
}
#endif
