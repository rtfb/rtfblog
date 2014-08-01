
This is an H1
=============

This is an H2
-------------

# This is an H1

## This is an H2

###### This is an H6

# This is an H1 #

## This is an H2 ##

### This is an H3 ######

- - -

Markdown uses email-style > characters for blockquoting. If you’re familiar
with quoting passages of text in an email message, then you know how to create
a blockquote in Markdown. It looks best if you hard wrap the text and put a >
before every line:

> This is a blockquote with two paragraphs. Lorem ipsum dolor sit amet,
> consectetuer adipiscing elit. Aliquam hendrerit mi posuere lectus.
> Vestibulum enim wisi, viverra nec, fringilla in, laoreet vitae, risus.
> 
> Donec sit amet nisl. Aliquam semper ipsum sit amet velit. Suspendisse
> id sem consectetuer libero luctus adipiscing.

Markdown allows you to be lazy and only put the > before the first line of a
hard-wrapped paragraph:

> This is a blockquote with two paragraphs. Lorem ipsum dolor sit amet,
consectetuer adipiscing elit. Aliquam hendrerit mi posuere lectus.
Vestibulum enim wisi, viverra nec, fringilla in, laoreet vitae, risus.

> Donec sit amet nisl. Aliquam semper ipsum sit amet velit. Suspendisse
id sem consectetuer libero luctus adipiscing.

Blockquotes can be nested (i.e. a blockquote-in-a-blockquote) by adding
additional levels of >:

> This is the first level of quoting.
>
> > This is nested blockquote.
>
> Back to the first level.

Blockquotes can contain other Markdown elements, including headers, lists, and
code blocks:

> ## This is a header.
> 
> 1.   This is the first list item.
> 2.   This is the second list item.
> 
> Here's some example code:
> 
>     return shell_exec("echo $input | $markdown_script");

- - -

Unordered lists use asterisks, pluses, and hyphens — interchangably — as list
markers:

*   Red
*   Green
*   Blue

is equivalent to:

+   Red
+   Green
+   Blue

and:

-   Red
-   Green
-   Blue

Ordered lists use numbers followed by periods:

1.  Bird
2.  McHale
3.  Parish

- - -

This is a normal paragraph:

    This is a code block.

Here is an example of AppleScript:

    tell application "Foo"
        beep
    end tell

- - -

This is [an example](http://example.com/ "Title") inline link.

[This link](http://example.net/) has no title attribute.

See my [About](/about/) page for details.

This is [an example][id] reference-style link.

You can optionally use a space to separate the sets of brackets:

This is [an example] [id] reference-style link.

[id]: http://example.com/  "Optional Title Here"

- - -

EMPHASIS

*single asterisks*

_single underscores_

**double asterisks**

__double underscores__

- - -

CODE

Use the `printf()` function.

``There is a literal backtick (`) here.``

A single backtick in a code span: `` ` ``

A backtick-delimited string in a code span: `` `foo` ``

