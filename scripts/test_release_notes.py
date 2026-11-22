import unittest

from release_notes import release_body


class ReleaseNotesTest(unittest.TestCase):
    notes = ("## [Unreleased]\n\n- Future\n\n## [1.0.0] - 2026-11-22\n\n"
             "### Changes\n\n- First\n\n```md\n## Example\n```\n\n"
             "## [0.9.0] - 2026-10-01\n\n- Older\n")

    def test_only_the_selected_markdown(self):
        self.assertEqual(release_body("v1.0.0", "v1.0.0\n", self.notes),
                         "### Changes\n\n- First\n\n```md\n## Example\n```\n")

    def test_invalid_inputs(self):
        for tag, version, notes in [
            ("v01.0.0", "v01.0.0", self.notes),
            ("1.0.0", "1.0.0", self.notes),
            ("v1.0.0", "v2.0.0", self.notes),
            ("v2.0.0", "v2.0.0", self.notes),
            ("v1.0.0", "v1.0.0", self.notes + "## [1.0.0] - 2026-11-23\n- Duplicate"),
            ("v1.0.0", "v1.0.0", "## [Unreleased]\n## [1.0.0] - 2026-11-22\n### Fixes\n<!-- TODO -->"),
            ("v1.0.0", "v1.0.0", self.notes.replace("2026-11-22", "2026-02-30")),
        ]:
            with self.subTest(tag=tag, version=version, notes=notes), self.assertRaises(ValueError):
                release_body(tag, version, notes)
