#FUNCIONES: las funcion son bloques de codigo que se pueden reutilizar (miralos como un portal que te lleva a otro lugar)

# SIempre la funciones empiezan con la palabra reservada def, luego el nombre de la funcion y entre parentesis los parametros que recibe,
#puede o no recibir parametros. Termina con ":".

def amo_a_mi_amor(nombre):
    print(f"Te amo mi amor: {nombre}")


#Para llamar una funcion solo tengo que hacerlo mediante el nombre de este y entre parentesisi los parametros segun como este la estructura de este

amo_a_mi_amor("Danisiwis")


#En el mismo orden en el que se define los parametros en la creacion de la funcion def(parametro1, paramereo2, ...), es el mismo orden
#en la que nosotro metemos los valores de esos parametros

def suma_division(a, b, c):
    d = (c + b)/a
    print(d)

suma_division(b = 3, c = 1, a = 5)

#Entramos a un conpeto importante: que es el scope 



def casita(persona):
    x = 5
    print(persona, x)

x = 10

print(x)
casita("Dani")

#AHora las funciones cuando las llamamos pueden DEVOLVER valores (osea no imprimir con print) si no que dar un valor segun los calculos
#o algoritmos que habia dentro de la funcion

def exponeciacion(a, b):
    return a**b
    # Despues de la ejecucion del return NO se ejecuta mas codigo
    nombre = johan
    print(johan)


y = exponeciacion(2, 3)
print(y)

print(exponeciacion(5, 12))

if exponeciacion(1, 3) > 5:
    print("Hola")


#CICLOS WHILE

#un ciclo lo que hace es que se ejecute un parte de un codigo muchas veces segun la condicion que este tenga.
#podrimos que si una codicion no se cumple, entonces la misma parte de codigo se ejecute y ejecutre hasta que se cumpla la funcion


parar = False
while parar == False:
    print("hola")
    num = int(input("Dame 1 para parar! "))
    if num == 1:
        parar = True

def exponeciacion(a, b):
    return a**b

a = int(input())
b = int(input())

while exponeciacion(a, b) <= 8:
    print("Mete otros numeros")
    a = int(input())
    b = int(input())


while True:
    print("Vas a estrar a las discotecas de Zona rosa!.")
    edad = int(input("Tu edad: "))
    if edad >= 18:
        print("Entra!")
        if edad >= 25:
            print("cerveza gratis")
            continue # Continue significa pasar a la siguiente iteracion (osea, se vuelve a ejecutar el bloque de codigo. OJO NO ROMPE NI NADA) sin importan si habia codigo abajo de este.
        break #ROMPER -> termino si o si el ciclo y todo el codigo abajo de este NO se ejecuta, es parecido al RETURN en las funciones
    else:
        print("No entra")
    print("Vuelva luego")


