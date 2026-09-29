import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock, Mock, patch

from bot import bark as bark_module


class FakeContent(dict):
    def __init__(self, body, reply_to=None):
        super().__init__(body=body, msgtype=bark_module.MessageType.TEXT)
        self.body = body
        self.msgtype = bark_module.MessageType.TEXT
        self._reply_to = reply_to

    def get_reply_to(self):
        return self._reply_to

    def trim_reply_fallback(self):
        if "\n\n" in self.body:
            self.body = self.body.rsplit("\n\n", 1)[1]


class FakeOutbound:
    def __init__(self, msgtype, body):
        self.msgtype = msgtype
        self.body = body
        self.reply_to = None

    def set_reply(self, event):
        self.reply_to = event


class BotLanguageTests(unittest.TestCase):
    def setUp(self):
        self.bot = bark_module.Bark.__new__(bark_module.Bark)
        self.bot.config = {
            "baas_url": "http://baas",
            "target_rooms": ["!room:example.org"],
            "detect_language": "fr",
            "response_language": "en",
            "room_languages": {},
        }
        self.bot._global_detect_language = "fr"
        self.bot._global_response_language = "en"
        self.bot._strict_min_score = 75
        self.bot._fuzzy_min_score = 55
        self.bot._embedded_min_coverage = 25
        self.bot._room_language_cache = {
            "!room:example.org": ("fr", "ja"),
        }
        self.bot.log = Mock()
        self.bot.client = SimpleNamespace(
            mxid="@bark:example.org",
            parse_user_id=lambda mxid: ("bark", "example.org"),
            send_message=AsyncMock(),
            get_event=AsyncMock(),
        )
        self.calls = []

        async def fake_get_json(url, path, params=None):
            self.calls.append((path, params))
            language = (params or {}).get("lang", "en")
            if path == "/isbark":
                text = (params or {}).get("text", "")
                is_bark = text in {"ouaf", "わん", "bark"}
                return {
                    "language": language,
                    "closest_language": language,
                    "family": "waf" if text == "ouaf" else "woof" if text == "わん" else "bark",
                    "match_type": "exact" if is_bark else "none",
                    "is_bark": is_bark,
                    "bark_percentage": "100.0%" if is_bark else "0.0%",
                }
            if path == "/barklike":
                text = (params or {}).get("text", "")
                embedded = text != "bark" and text.endswith("bark")
                if language == "auto":
                    language = {
                        "ouaf": "fr",
                        "hav": "tr",
                        "гав": "ru",
                        "дзяв": "uk",
                        "汪汪": "zh-Hans",
                        "bark": "en",
                    }.get(text, "en" if embedded else "auto")
                return {
                    "language": language,
                    "closest_language": language,
                    "family": {
                        "ouaf": "waf", "hav": "waf", "дзяв": "waf",
                        "わん": "woof",
                    }.get(text, "bark"),
                    "is_bark_like": text in {
                        "ouaf", "わん", "hav", "гав", "дзяв", "汪汪", "bark"
                    } or embedded,
                    "bark_percentage": "30.8%" if embedded else "100.0%",
                    "match_type": "embedded" if embedded else "exact",
                    "matched_text": "bark" if embedded else None,
                    "coverage_percentage": 30.8 if embedded else 100,
                }
            if path == "/bark":
                sounds = {
                    "fr": "ouaf", "ja": "わん", "tr": "hav",
                    "ru": "гав", "uk": "дзяв", "zh-hans": "汪汪",
                    "hy": "հավ",
                }
                return {"language": language, "bark": sounds.get(language.lower(), "bark")}
            if path == "/askbark":
                return {"answer": "Answer."}
            raise AssertionError(f"unexpected BaaS path: {path}")

        self.bot._baas_get_json = AsyncMock(side_effect=fake_get_json)
        self.patch_outbound = patch.object(
            bark_module, "TextMessageEventContent", FakeOutbound
        )
        self.patch_outbound.start()
        self.addCleanup(self.patch_outbound.stop)

    def event(self, body, room="!room:example.org", reply_to=None, sender="@alice:example.org"):
        return SimpleNamespace(
            sender=sender,
            room_id=room,
            content=FakeContent(body, reply_to=reply_to),
            reply=AsyncMock(),
        )

    def sent_body(self):
        content = self.bot.client.send_message.await_args.args[1]
        return content.body

    def test_options_are_leading_and_one_time(self):
        remaining, options, error = bark_module.Bark._parse_language_options(
            " --detect=FR --reply=ja You’re cute bark"
        )
        self.assertEqual(remaining, "You’re cute bark")
        self.assertEqual(options, {"detect": "fr", "response": "ja"})
        self.assertIsNone(error)

        remaining, options, error = bark_module.Bark._parse_language_options(
            "bark --reply=ja"
        )
        self.assertEqual(remaining, "bark --reply=ja")
        self.assertEqual(options, {})
        self.assertIsNone(error)

        _, _, duplicate = bark_module.Bark._parse_language_options(
            "--reply=fr --reply=ja bark"
        )
        self.assertIsNotNone(duplicate)
        _, _, invalid = bark_module.Bark._parse_language_options("--detect=xx bark")
        self.assertEqual(invalid, "language:xx")

        remaining, options, error = bark_module.Bark._parse_language_options(
            "--detect=ZH-Hant --reply=hy 汪汪"
        )
        self.assertEqual(remaining, "汪汪")
        self.assertEqual(options, {"detect": "zh-hant", "response": "hy"})
        self.assertIsNone(error)

        remaining, options, error = bark_module.Bark._parse_language_options(
            "--detect=auto --reply=auto bark"
        )
        self.assertEqual(
            (remaining, options, error),
            ("bark", {"detect": "auto", "response": "auto"}, None),
        )

    def test_default_language_settings_are_auto(self):
        class Config(dict):
            def load_and_update(self):
                pass

        self.bot.config = Config(baas_url="http://baas", target_rooms=["!room:example.org"])

        import asyncio

        asyncio.run(self.bot.start())

        self.assertEqual(self.bot._resolve_languages("!room:example.org"), ("auto", "auto"))

        asyncio.run(
            bark_module.Bark.handle_message(self.bot, self.event("@bark:example.org hav"))
        )
        self.assertIn(("/isbark", {"text": "hav", "lang": "auto"}), self.calls)
        self.assertIn(("/barklike", {"text": "hav", "lang": "auto"}), self.calls)
        self.assertIn("Verdict:", self.sent_body())

        self.calls.clear()
        self.bot.client.send_message.reset_mock()
        asyncio.run(
            bark_module.Bark.handle_message(self.bot, self.event("@bark:example.org"))
        )
        self.assertEqual(self.calls, [("/bark", {"lang": "en"})])

    def test_help_command_is_english_in_every_room(self):
        import asyncio

        asyncio.run(
            bark_module.Bark.handle_message(
                self.bot, self.event("@bark:example.org help")
            )
        )
        self.assertIn("Bark Help:", self.sent_body())
        self.assertFalse(self.calls)

    def test_room_override_is_independent(self):
        self.assertEqual(
            self.bot._resolve_languages("!room:example.org"), ("fr", "ja")
        )
        self.assertEqual(
            self.bot._resolve_languages("!other:example.org"), ("fr", "en")
        )

    def test_embedded_analysis_uses_english_diagnostic(self):
        strict = {"bark_percentage": "0.0%", "is_bark": False, "language": "auto"}
        fuzzy = {
            "bark_percentage": "30.8%",
            "is_bark_like": True,
            "match_type": "embedded",
            "matched_text": "bark",
            "coverage_percentage": 30.8,
            "language": "en",
            "family": "bark",
        }
        text, formatted = self.bot._format_analysis(strict, fuzzy)
        self.assertIn("Verdict: Bark-like.", text)
        self.assertIn("Language: en", text)
        self.assertIn("Sound: bark · embedded", text)
        self.assertIn("Strict: 0.0% · Fuzzy: 30.8%", text)
        self.assertIn("Matched: 'bark' · 30.8% of letters", text)
        self.assertIn(
            '<strong><span data-mx-color="#E1B97D">Verdict: Bark-like.</span></strong>',
            formatted,
        )
        self.assertIn('<span data-mx-color="#D98792">0.0%</span>', formatted)
        self.assertIn('30.8%</span> of letters', formatted)

    def test_shared_score_and_metadata_are_shown_once(self):
        strict = {
            "is_bark": True, "bark_percentage": "100.0%", "language": "en",
            "family": "bark", "match_type": "exact",
        }
        fuzzy = {
            **strict, "is_bark_like": True,
        }
        text, formatted = self.bot._format_analysis(strict, fuzzy)
        self.assertEqual(
            text,
            "Verdict: Bark.\nLanguage: en\nSound: bark · exact\n"
            "Strict / fuzzy score: 100.0%\nLimits: strict ≥75%, fuzzy ≥55%",
        )
        self.assertIn(
            '<strong><span data-mx-color="#7FC4A3">Verdict: Bark.</span></strong>',
            formatted,
        )
        self.assertIn('<span data-mx-color="#7FC4A3">100.0%</span>', formatted)

        fuzzy["bark_percentage"] = "85.0%"
        text, formatted = self.bot._format_analysis(strict, fuzzy)
        self.assertIn("Strict: 100.0% · Fuzzy: 85.0%", text)
        self.assertIn('<span data-mx-color="#9CC198">85.0%</span>', formatted)

    def test_analysis_footer_is_escaped_html_with_plain_text_fallback(self):
        strict = {"is_bark": False, "bark_percentage": "0.0%", "language": "auto"}
        fuzzy = {
            "is_bark_like": True, "bark_percentage": "50.0%",
            "language": "en", "family": "bark", "match_type": "embedded",
            "matched_text": "<bark & wow>", "coverage_percentage": 50,
            "squeezed_form": "<bark & wow>", "detection_time": "12µs",
        }
        body, formatted = self.bot._format_analysis(strict, fuzzy)
        self.assertIn("Normalized: <bark & wow> · API: fuzzy 12µs", body)
        self.assertIn("Limits: strict ≥75%, coverage ≥25%", body)
        self.assertIn("<sub><span", formatted)
        self.assertIn('<span data-mx-color="#E1B97D">50.0%</span>', formatted)
        self.assertIn("&lt;bark &amp; wow&gt;", formatted)
        self.assertNotIn("<bark & wow>", formatted)

    def test_configurable_thresholds_control_verdicts_and_passive_replies(self):
        import asyncio

        strict = {
            "is_bark": True, "bark_percentage": "80.0%", "language": "auto",
            "closest_language": "en", "family": "bark", "match_type": "typo",
        }
        fuzzy = {
            "is_bark_like": True, "bark_percentage": "60.0%",
            "language": "auto", "closest_language": "en", "family": "bark",
            "match_type": "typo",
        }
        self.bot._strict_min_score = 85
        self.bot._fuzzy_min_score = 65
        text, formatted = self.bot._format_analysis(strict, fuzzy)
        self.assertIn("Language: undetermined (closest: en)", text)
        self.assertIn("Verdict: Not bark.", text)
        self.assertIn(
            '<strong><span data-mx-color="#D98792">Verdict: Not bark.</span></strong>',
            formatted,
        )

        self.bot._embedded_min_coverage = 35
        evt = self.event("You're cute bark")
        asyncio.run(bark_module.Bark.handle_message(self.bot, evt))
        evt.reply.assert_not_awaited()

        embedded = {
            "is_bark_like": True, "bark_percentage": "30.8%",
            "match_type": "embedded", "coverage_percentage": 30.8,
        }
        self.assertFalse(self.bot._matches_fuzzy(embedded))

    def test_threshold_config_uses_api_floors(self):
        self.bot.config.update(
            strict_min_score=10, fuzzy_min_score="90", embedded_min_coverage="no"
        )
        self.assertEqual(self.bot._valid_threshold("strict_min_score", 75), 75)
        self.assertEqual(self.bot._valid_threshold("fuzzy_min_score", 55), 90)
        self.assertEqual(self.bot._valid_threshold("embedded_min_coverage", 25), 25)

    def test_ping_detects_in_french_and_keeps_analysis_english(self):
        async def run():
            await bark_module.Bark.handle_message(
                self.bot, self.event("@bark:example.org --detect=fr --reply=ja ouaf")
            )

        import asyncio

        asyncio.run(run())
        self.assertEqual(self.calls[0], ("/isbark", {"text": "ouaf", "lang": "fr"}))
        self.assertEqual(self.calls[1], ("/barklike", {"text": "ouaf", "lang": "fr"}))
        self.assertIn("Verdict: Bark.", self.sent_body())
        content = self.bot.client.send_message.await_args.args[1]
        self.assertEqual(content.format, bark_module.Format.HTML)
        self.assertIn(
            '<strong><span data-mx-color="#7FC4A3">Verdict: Bark.</span></strong>',
            content.formatted_body,
        )

    def test_bare_ping_generates_using_room_response_language(self):
        import asyncio

        asyncio.run(
            bark_module.Bark.handle_message(
                self.bot, self.event("@bark:example.org")
            )
        )
        self.assertEqual(self.calls, [("/bark", {"lang": "ja"})])
        self.assertEqual(self.sent_body(), "わん")

    def test_per_ping_response_can_select_new_language(self):
        import asyncio

        asyncio.run(
            bark_module.Bark.handle_message(
                self.bot, self.event("@bark:example.org --reply=hy")
            )
        )
        self.assertEqual(self.calls, [("/bark", {"lang": "hy"})])

    def test_fact_check_stays_english_with_reply_language(self):
        import asyncio

        asyncio.run(
            bark_module.Bark.handle_message(
                self.bot,
                self.event("@bark:example.org --reply=ja is this true: dogs bark"),
            )
        )
        path, params = self.calls[-1]
        self.assertEqual(path, "/askbark")
        self.assertEqual(params, {"text": "dogs bark [@alice:example.org]"})
        self.assertEqual(self.sent_body(), "Answer.")

    def test_invalid_option_gets_english_usage_without_api_call(self):
        import asyncio

        asyncio.run(
            bark_module.Bark.handle_message(
                self.bot, self.event("@bark:example.org --detect=xx ouaf")
            )
        )
        self.assertFalse(self.calls)
        self.assertIn("Unsupported language", self.sent_body())
        self.assertIn("--detect", self.sent_body())

    def test_passive_room_auto_detects_and_replies_in_the_detected_language(self):
        import asyncio

        async def run():
            for form, language in (
                ("ouaf", "fr"),
                ("hav", "tr"),
                ("гав", "ru"),
                ("дзяв", "uk"),
                ("汪汪", "zh-Hans"),
            ):
                evt = self.event(form)
                await bark_module.Bark.handle_message(self.bot, evt)
                self.assertIn(("/barklike", {"text": form, "lang": "auto"}), self.calls)
                self.assertIn(("/bark", {"lang": language}), self.calls)
                evt.reply.assert_awaited_once()

        asyncio.run(run())

    def test_reply_fallback_text_is_trimmed_before_detection(self):
        import asyncio

        parent = SimpleNamespace(
            sender="@bob:example.org",
            content=FakeContent("> quoted old text\n\nouaf"),
        )
        self.bot.client.get_event.return_value = parent
        evt = self.event(
            "@bark:example.org --detect=fr --reply=ja ouaf?", reply_to="$parent"
        )
        self.assertEqual(evt.content.get_reply_to(), "$parent")
        asyncio.run(
            bark_module.Bark.handle_message(self.bot, evt)
        )
        self.bot.client.get_event.assert_awaited_once_with(evt.room_id, "$parent")
        self.assertIn(("/isbark", {"text": "ouaf", "lang": "fr"}), self.calls)
        self.assertIn(
            ("/askbark", {"text": "ouaf [@bob:example.org]"}),
            self.calls,
        )

    def test_edit_events_are_ignored(self):
        import asyncio

        evt = self.event("ouaf")
        evt.content["m.relates_to"] = {"rel_type": "m.replace"}
        asyncio.run(bark_module.Bark.handle_message(self.bot, evt))
        self.assertFalse(self.calls)

    def test_older_api_without_language_field_is_actionable(self):
        import asyncio

        class Response:
            async def __aenter__(self):
                return self

            async def __aexit__(self, exc_type, exc, traceback):
                return False

            def raise_for_status(self):
                return None

            async def json(self):
                return {"bark": "bark"}

        self.bot.http = SimpleNamespace(get=Mock(return_value=Response()))

        async def request():
            with self.assertRaises(bark_module.UnsupportedLanguageError) as raised:
                await bark_module.Bark._baas_get_json(
                    self.bot, "http://baas", "/bark", {"lang": "fr"}
                )
            self.assertEqual(raised.exception.language, "fr")

        asyncio.run(request())

    def test_auto_detection_accepts_the_detected_canonical_language(self):
        import asyncio

        class Response:
            async def __aenter__(self):
                return self

            async def __aexit__(self, exc_type, exc, traceback):
                return False

            def raise_for_status(self):
                return None

            async def json(self):
                return {"language": "zh-Hans", "is_bark_like": True}

        self.bot.http = SimpleNamespace(get=Mock(return_value=Response()))

        async def request():
            data = await bark_module.Bark._baas_get_json(
                self.bot,
                "http://baas",
                "/barklike",
                {"text": "汪汪", "lang": "auto"},
            )
            self.assertEqual(data["language"], "zh-Hans")

        asyncio.run(request())


if __name__ == "__main__":
    unittest.main()
