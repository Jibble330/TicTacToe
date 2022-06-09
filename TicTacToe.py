import pygame
import numpy as np
import ctypes
import sys
import socket
import threading

user32 = ctypes.windll.user32
user32.SetProcessDPIAware()

SCREEN_WIDTH = user32.GetSystemMetrics(0)
SCREEN_HEIGHT = user32.GetSystemMetrics(1)
FPS = 60
WHITE = (255, 255, 255)
BACKGROUND = ()

def event_handler():
    for event in pygame.event.get():
        if event.type == pygame.KEYDOWN:
            if event.key == pygame.K_ESCAPE:
                sys.exit()
        if event.type == pygame.MOUSEBUTTONDOWN:
            mouseX, mouseY = pygame.mouse.get_pos()

def main():
    screen = pygame.display.set_mode((SCREEN_WIDTH, SCREEN_HEIGHT))
    pygame.display.set_caption("Tic-Tac-Toe")
    clock = pygame.time.Clock()
    tiles = np.full([3, 3], 0)
    while True:
        event_handler()
        screen.fill(WHITE)
        pygame.display.flip()
        clock.tick(FPS)

if __name__ == "__main__":
    main()