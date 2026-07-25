import unittest

import compare


class ParserTests(unittest.TestCase):
    def test_parse_volt(self):
        result = compare.parse_volt(
            '{"summary":{"completedRequests":1200,"throughput":599.25}}'
        )
        self.assertEqual(result, compare.ToolResult(1200, 599.25))

    def test_parse_wrk(self):
        result = compare.parse_wrk(
            """
Running 2s test @ http://target:8080/empty
  4 threads and 10 connections
  24690 requests in 2.00s, 2.40MB read
Requests/sec:  12345.00
Transfer/sec:      1.20MB
"""
        )
        self.assertEqual(result, compare.ToolResult(24690, 12345.0))

    def test_parse_hey_sums_status_counts(self):
        result = compare.parse_hey(
            """
Summary:
  Requests/sec: 1000.50

Status code distribution:
  [200] 1990 responses
  [500] 10 responses
"""
        )
        self.assertEqual(result, compare.ToolResult(2000, 1000.5))

    def test_parse_k6(self):
        result = compare.parse_k6(
            '{"requests":3456,"requests_per_second":1728.0}'
        )
        self.assertEqual(result, compare.ToolResult(3456, 1728.0))

    def test_malformed_output_is_rejected(self):
        parsers = (
            compare.parse_volt,
            compare.parse_wrk,
            compare.parse_hey,
            compare.parse_k6,
        )
        for parser in parsers:
            with self.subTest(parser=parser.__name__):
                with self.assertRaises(ValueError):
                    parser("not valid output")


class ConfigurationTests(unittest.TestCase):
    def test_parse_concurrency(self):
        self.assertEqual(compare.parse_concurrency("10, 50,100"), [10, 50, 100])

    def test_parse_concurrency_rejects_non_positive_values(self):
        for raw in ("", "0", "10,-1", "ten"):
            with self.subTest(raw=raw):
                with self.assertRaises(ValueError):
                    compare.parse_concurrency(raw)

    def test_request_count_allows_bounded_in_flight_requests(self):
        self.assertEqual(compare.validate_request_count(100, 100, 10), 0)
        self.assertEqual(compare.validate_request_count(100, 110, 10), 10)

    def test_request_count_rejects_impossible_values(self):
        for reported, observed in ((100, 99), (100, 111)):
            with self.subTest(reported=reported, observed=observed):
                with self.assertRaises(RuntimeError):
                    compare.validate_request_count(reported, observed, 10)

    def test_tool_name_is_removed_from_version_output(self):
        self.assertEqual(
            compare.normalize_version(
                "k6", "k6 v2.0.0 (go1.25.4, linux/arm64)\n", "fallback"
            ),
            "v2.0.0 (go1.25.4, linux/arm64)",
        )
        self.assertEqual(
            compare.normalize_version("hey", "0.1.5\n", "fallback"),
            "0.1.5",
        )
        self.assertEqual(compare.normalize_version("wrk", "", "4.2.0"), "4.2.0")


class MarkdownTests(unittest.TestCase):
    def test_markdown_is_paste_ready(self):
        result = compare.ScenarioResult(
            workload="Empty response",
            concurrency=10,
            medians={"volt": 1000, "wrk": 900, "hey": 800, "k6": 700},
        )
        output = compare.markdown(
            [result],
            {
                "date": "2026-07-25",
                "duration": "10s",
                "runs": 5,
                "seed": 20260725,
                "host_os": "Linux arm64",
                "host_cpu": "Test CPU",
                "docker": "28.3.2",
                "volt_version": "abc123",
                "wrk_version": "4.2.0",
                "hey_version": "0.1.5",
                "k6_version": "2.0.0",
            },
        )

        self.assertTrue(
            output.startswith("<!-- BEGIN GENERATED BENCHMARK COMPARISON -->\n")
        )
        self.assertIn("| Empty response | 10 | 1,000 | 900 | 800 | 700 |", output)
        self.assertTrue(
            output.endswith("<!-- END GENERATED BENCHMARK COMPARISON -->\n")
        )


if __name__ == "__main__":
    unittest.main()
