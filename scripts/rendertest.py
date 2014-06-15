#!/usr/bin/env python

# These need to be installed for this to work:
#
# $ sudo pip install -U selenium
# $ sudo pip install pyvirtualdisplay
# $ sudo apt-get install xvfb

from pyvirtualdisplay import Display
from selenium import webdriver

display = Display(visible=0, size=(1600, 900))
display.start()

browser = webdriver.Firefox()

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

for u in urls:
    browser.get('http://localhost:8080/' + u)
    browser.save_screenshot(u + '.png')

browser.close()
display.stop()
