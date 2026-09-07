project = "Mykrobe2"
copyright = "2026, Martin Hunt"
author = "Martin Hunt"

extensions = [
    "myst_parser",
]

source_suffix = {
    ".md": "markdown",
}

master_doc = "index"

exclude_patterns = [
    "_build",
    "Thumbs.db",
    ".DS_Store",
]

html_theme = "furo"
html_title = "Mykrobe2"
html_theme_options = {
    "source_repository": "https://github.com/Mykrobe-tools/mykrobe2/",
    "source_branch": "main",
    "source_directory": "docs/",
    "top_of_page_buttons": ["view"],
}
