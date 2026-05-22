"""Tests for ARIZE_INSTALL_* env-var contract on prompt_* functions."""

from __future__ import annotations

import os

import pytest


@pytest.fixture(autouse=True)
def _scrub_install_env(monkeypatch):
    """Ensure ARIZE_INSTALL_* env vars are unset at test entry."""
    for key in list(os.environ):
        if key.startswith("ARIZE_INSTALL_"):
            monkeypatch.delenv(key, raising=False)
    yield


class TestPromptProjectName:
    def test_env_var_set_skips_prompt(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_PROJECT_NAME", "my-project")
        from core.setup import prompt_project_name

        assert prompt_project_name("default-name") == "my-project"

    def test_unset_non_interactive_returns_default(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_NON_INTERACTIVE", "1")
        from core.setup import prompt_project_name

        assert prompt_project_name("default-name") == "default-name"


class TestPromptUserId:
    def test_env_var_set_skips_prompt(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_USER_ID", "alice")
        from core.setup import prompt_user_id

        assert prompt_user_id() == "alice"

    def test_unset_non_interactive_returns_empty(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_NON_INTERACTIVE", "1")
        from core.setup import prompt_user_id

        assert prompt_user_id() == ""


class TestPromptContentLogging:
    def test_all_env_vars_skip_prompt(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_LOG_PROMPTS", "false")
        monkeypatch.setenv("ARIZE_INSTALL_LOG_TOOL_DETAILS", "true")
        monkeypatch.setenv("ARIZE_INSTALL_LOG_TOOL_CONTENT", "false")
        from core.setup import prompt_content_logging

        result = prompt_content_logging()
        assert result == {"prompts": False, "tool_details": True, "tool_content": False}

    def test_non_interactive_defaults_true(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_NON_INTERACTIVE", "1")
        from core.setup import prompt_content_logging

        result = prompt_content_logging()
        assert result == {"prompts": True, "tool_details": True, "tool_content": True}


class TestPromptVerbose:
    def test_env_var_true(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_VERBOSE", "true")
        from core.setup import prompt_verbose

        assert prompt_verbose() is True

    def test_env_var_false(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_VERBOSE", "false")
        from core.setup import prompt_verbose

        assert prompt_verbose() is False

    def test_non_interactive_defaults_false(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_NON_INTERACTIVE", "1")
        from core.setup import prompt_verbose

        assert prompt_verbose() is False


class TestPromptBackend:
    def test_unset_non_interactive_errors(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_NON_INTERACTIVE", "1")
        from core.setup import prompt_backend

        with pytest.raises(SystemExit):
            prompt_backend(existing_harnesses=None)

    def test_invalid_backend_errors(self, monkeypatch):
        monkeypatch.setenv("ARIZE_INSTALL_BACKEND", "bogus")
        from core.setup import prompt_backend

        with pytest.raises(SystemExit):
            prompt_backend(existing_harnesses=None)
