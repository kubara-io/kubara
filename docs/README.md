# Introduction 
https://www.mkdocs.org/getting-started/

# Getting Started
Requires Python version 3.14.3 (pyenv install 3.14.3)
Its recommended to use pipenv for development.


## change directory to mkdocs directory
```bash
cd docs
```

## install pipenv
```bash
pip3 install pipenv
```

## setup virtual env
```bash
pipenv --python 3.14.3
```

## install dependencies from Pipfile
```bash
pipenv install
```

## start shell in virtual env
```bash
pipenv shell
```

## run mkdocs
```bash
mkdocs serve
```

## see live rendering
http://127.0.0.1:8000/


## Additional Styling Options:
https://squidfunk.github.io/mkdocs-material/reference/


### used dependencies
- mkdocs
- mkdocs-material
- mkdocs-macros-plugin

