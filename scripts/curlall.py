#!/usr/bin/env python

import urllib2


urls = ['serialas-mokykla-pirmoji-serija',
    'serialas-mokykla-antroji-serija',
    'serialas-mokykla-trecioji-serija',
    'fp-precision',
    'evoliucija',
    'softo-katsparumas',
    'serialas-mokykla-wrap-up',
    'mkdir',
    'quantum-bug-o-mechanics',
    'kaip-tapti-hakeriu',
    'plaukimas-sirvinta',
    'arrr',
    'kaip-nereikia-programuoti-dublis-du',
    'persikrausciau',
    'komentarai-laikinai-isjungti',
    'uzreferenduma-lt-the-good-the-bad-and-the-ugly',
    'intuicijos-pinkl-s',
    'atradimai-knygose',
    'vesnotas-1-8',
    'vim-dox-spell']


def write_file(name, content):
    f = open(name, 'w')
    f.write(content)
    f.close()


def curl(url):
    f = urllib2.urlopen(url)
    return f.read()


for u in urls:
    write_file(u + '.html', curl('http://localhost:8080/' + u))
