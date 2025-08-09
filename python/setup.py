#!/usr/bin/env python
"""
Setup script for gonka-openai
"""

from setuptools import setup

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="gonka-openai",
    version="0.2.2",
    author="David Liberman",
    author_email="david@liberman.net",
    description="OpenAI client with Gonka network integration",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/product-science/gonka-openai",
    packages=["gonka_openai"],
    package_dir={"gonka_openai": "src"},
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.7",
    install_requires=[
        "openai>=1.0.0",
        "python-dotenv>=0.19.0",
        "secp256k1>=0.14.0",
        "ecdsa>=0.18.0",
        "bech32>=1.2.0",
        "requests>=2.25.0",
        "protobuf>=4.25.0",
    ],
    extras_require={
        "dev": [
            "pytest>=6.0.0",
            "black>=21.5b2",
            "isort>=5.9.1",
            "mypy>=0.812",
        ],
    },
) 