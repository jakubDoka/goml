# goml

goml is a html-like "language" parser, module does not focus on rendering, only translating goml into useful data-structure. its very similar to html though syntax is stricter and also supports some additional features. JavaScript is not supported. 

## Showcase

```
<#>prefab definition<#>
<!yes_no>
    <div> 
        <button onclick={yes}>yes</>
        <button onclick={no}>no</>
    </>
<!/>

<div>Hello, is monday today?</>
<#>prefab used as any element, arguments will get substituted by given attributes<#>
<yes_no yes="yes-handler-link" no="no-handler-link"/>
```

If we pass following "code" to parse it will return (result is formatted by json.MarshallIdent so it is more readable)

```json
{
    "Name": "",
    "Attributes": {},
    "Children": [
        {
            "Name": "div",
            "Attributes": {},
            "Children": [
                {
                    "Name": "text",
                    "Attributes": {
                        "text": [
                            "Hello, is monday today?"
                        ]
                    },
                    "Children": null
                }
            ]
        },
        {
            "Name": "div",
            "Attributes": {},
            "Children": [
                {
                    "Name": "button",
                    "Attributes": {
                        "onclick": [
                            "yes-handler-link"
                        ]
                    },
                    "Children": [
                        {
                            "Name": "text",
                            "Attributes": {
                                "text": [
                                    "yes"
                                ]
                            },
                            "Children": []
                        }
                    ]
                },
                {
                    "Name": "button",
                    "Attributes": {
                        "onclick": [
                            "no-handler-link"
                        ]
                    },
                    "Children": [
                        {
                            "Name": "text",
                            "Attributes": {
                                "text": [
                                    "no"
                                ]
                            },
                            "Children": []
                        }
                    ]
                }
            ]
        }
    ]
}
```

As you can see hard-coding same thing in go can be unpleasant, goml syntax is straight forward and lot shorter, it also has jsx like features(i call them prefabs). Not this may well be absolutely useless to you though i made goml as dependency of gobatch(in progress) for ui framework. 

## Syntax

I tried to make error messages as convenient as possible though i still think that documenting the "language" this way is necessary so lets go over it.

Simplest thing you can do is `<div/>`, all this does is creating element with no attributes and no children to a root element, of corse if `div` is not added with `goml.Parser.AddDefinitions()`, error reporting unknown element will be returned. 

### Attributes

There are 3 ways of defining attributes. Writing `<div boolean hello="hello" list=["first" "second" "last"]/>` covers all the syntax outside prefab definition(to that later). Go form of attributes will look like:

```go
map[string][]string {
    "boolean": {"true"},
    "hello":   {"hello"},
    "list":    {"first", "second", "last"},
}
```

The way example is written is only way to write it(other then order of attributes(witch does not matter)). No extra spaces nor characters are allowed, try adding random space not including inside strings and parsing will fail. List also has no commas because they would be useless parsing overhead as all you can put in there are string.

### Nesting and Comments

This should be self explanatory, don't forget to close every comment. The spaces, tabs and new lines does not matter. also to close element you don't have to write just '</>' and it will close currently open element. Repeating name in closure is (in my modest opinion) useless as it jus creates space for incorrect syntax like 
`<div><button></div></button>`.
```
<#> this is mess <#>
<div><div><div><div><div><div></></></></></></>

<#> this is less messy but still does the same thing <#>
<div>
    <div>
        <div>
            <div>
                <div>
                    <div></>
                </>
            </>
        </>
    </>
</>
```

### Prefabs and Text

Prefabs are probably best feature of goml which html should have. Prefab is something like a template you can reuse after definition. You can put template marks on four places. You can use marker as value, part of value, element of list, part of element of list and in body text. Behavior of test is similar of that in html, multiple spaces and newlines are truncated, after space and before space is also ignored. to use tab or newline you have to explicitly write it. This allows you format your code freely.

```
<!prefab>
    <div name={name} description="greeting of {name}" useless=[{one} "{two} three"]> 
        Hello {name}...\n
        How are you?
    </>
<!/>

<#> using a prefab <#>
<prefab name="idk" name="Joe" one="whatever" two="and more"/>

<#> is equivalent to <#>
<div name="idk" description="greeting of Joe" useless=["whatever" "and more three"]> 
    Hello Joe...\n
    How are you?
</>
```

If you need extra spaces you can use `\` to prefix space so it will not get truncated. Same goes for writhing `<`, you have to write `\<` or it will be considered a new element. Mind that text will be parsed into element with name `text` and attribute `text` where string is stored. 